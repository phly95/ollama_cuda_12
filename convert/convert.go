package convert

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/protobuf/proto"

	"github.com/jmorganca/ollama/convert/sentencepiece"
	"github.com/jmorganca/ollama/llm"
)

type Params struct {
	Architectures       []string `json:"architectures"`
	VocabSize           int      `json:"vocab_size"`
	HiddenSize          int      `json:"hidden_size"`       // n_embd
	HiddenLayers        int      `json:"num_hidden_layers"` // n_layer
	ContextSize         int      `json:"max_position_embeddings"`
	IntermediateSize    int      `json:"intermediate_size"`
	AttentionHeads      int      `json:"num_attention_heads"` // n_head
	KeyValHeads         int      `json:"num_key_value_heads"`
	NormEPS             float64  `json:"rms_norm_eps"`
	RopeFreqBase        float64  `json:"rope_theta"`
	BoSTokenID          int      `json:"bos_token_id"`
	EoSTokenID          int      `json:"eos_token_id"`
	HeadDimension       int      `json:"head_dim"`
	PaddingTokenID      int      `json:"pad_token_id"`
	UseParallelResidual bool     `json:"use_parallel_residual"`
}

type MetaData struct {
	Type    string `mapstructure:"dtype"`
	Shape   []int  `mapstructure:"shape"`
	Offsets []int  `mapstructure:"data_offsets"`
}

func ReadSafeTensors(fn string, offset uint64) ([]llm.Tensor, uint64, error) {
	f, err := os.Open(fn)
	if err != nil {
		return []llm.Tensor{}, 0, err
	}
	defer f.Close()

	var jsonSize uint64
	binary.Read(f, binary.LittleEndian, &jsonSize)

	buf := make([]byte, jsonSize)
	_, err = io.ReadFull(f, buf)
	if err != nil {
		return []llm.Tensor{}, 0, err
	}

	d := json.NewDecoder(bytes.NewBuffer(buf))
	d.UseNumber()
	var parsed map[string]interface{}
	if err = d.Decode(&parsed); err != nil {
		return []llm.Tensor{}, 0, err
	}

	var keys []string
	for k := range parsed {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	slog.Info("converting layers")

	var tensors []llm.Tensor
	for _, k := range keys {
		vals := parsed[k].(map[string]interface{})
		var data MetaData
		if err = mapstructure.Decode(vals, &data); err != nil {
			return []llm.Tensor{}, 0, err
		}

		var size uint64
		var kind uint32
		switch len(data.Shape) {
		case 0:
			// metadata
			continue
		case 1:
			// convert to float32
			kind = 0
			size = uint64(data.Shape[0] * 4)
		case 2:
			// convert to float16
			kind = 1
			size = uint64(data.Shape[0] * data.Shape[1] * 2)
		}

		ggufName, err := GetTensorName(k)
		if err != nil {
			slog.Error("%v", err)
			return []llm.Tensor{}, 0, err
		}

		shape := []uint64{0, 0, 0, 0}
		for i := range data.Shape {
			shape[i] = uint64(data.Shape[i])
		}

		t := llm.Tensor{
			Name:          ggufName,
			Kind:          kind,
			Offset:        offset,
			Shape:         shape[:],
			FileName:      fn,
			OffsetPadding: 8 + jsonSize,
			FileOffsets:   []uint64{uint64(data.Offsets[0]), uint64(data.Offsets[1])},
		}
		slog.Debug(fmt.Sprintf("%+v", t))
		tensors = append(tensors, t)
		offset += size
	}
	return tensors, offset, nil
}

func GetSafeTensors(dirpath string) ([]llm.Tensor, error) {
	var tensors []llm.Tensor
	files, err := filepath.Glob(filepath.Join(dirpath, "/model-*.safetensors"))
	if err != nil {
		return []llm.Tensor{}, err
	}

	var offset uint64
	for _, f := range files {
		var t []llm.Tensor
		var err error
		t, offset, err = ReadSafeTensors(f, offset)
		if err != nil {
			slog.Error("%v", err)
			return []llm.Tensor{}, err
		}
		tensors = append(tensors, t...)
	}
	return tensors, nil
}

func GetParams(dirpath string) (*Params, error) {
	f, err := os.Open(filepath.Join(dirpath, "config.json"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var params Params

	d := json.NewDecoder(f)
	err = d.Decode(&params)
	if err != nil {
		return nil, err
	}

	return &params, nil
}

// Details on gguf's tokenizer can be found at:
// https://github.com/ggerganov/ggml/blob/master/docs/gguf.md#tokenizer
type Vocab struct {
	Tokens []string
	Scores []float32
	Types  []int32
	Merges []string
}

const (
	TokenTypeUndefined   int32 = 0
	TokenTypeNormal      int32 = 1
	TokenTypeUnknown     int32 = 2
	TokenTypeControl     int32 = 3
	TokenTypeUserDefined int32 = 4
	TokenTypeUnused      int32 = 5
	TokenTypeByte        int32 = 6
)

func LoadTokens(dirpath string) (*Vocab, error) {
	slog.Info(fmt.Sprintf("reading vocab from %s", filepath.Join(dirpath, "tokenizer.model")))
	in, err := os.ReadFile(filepath.Join(dirpath, "tokenizer.model"))
	if err != nil {
		slog.Debug("XXX got err: ", "error", err)
		if errors.Is(err, fs.ErrNotExist) {
			return loadJsonTokens(dirpath)
		}
		return nil, err
	}

	// To regenerate sentencepiece from the protobufs use:
	// protoc -I=./ --go_out=./ sentencepiece_model.proto
	modelProto := &sentencepiece.ModelProto{}
	if err := proto.Unmarshal(in, modelProto); err != nil {
		return nil, err
	}

	v := &Vocab{
		Tokens: make([]string, 0),
		Scores: make([]float32, 0),
		Types:  make([]int32, 0),
	}

	pieces := modelProto.GetPieces()
	for _, p := range pieces {
		v.Tokens = append(v.Tokens, p.GetPiece())
		v.Scores = append(v.Scores, p.GetScore())
		t := p.GetType()
		switch t {
		case sentencepiece.ModelProto_SentencePiece_UNKNOWN:
		case sentencepiece.ModelProto_SentencePiece_CONTROL:
		case sentencepiece.ModelProto_SentencePiece_UNUSED:
		case sentencepiece.ModelProto_SentencePiece_BYTE:
		default:
			t = sentencepiece.ModelProto_SentencePiece_NORMAL
		}
		v.Types = append(v.Types, int32(t))
	}

	slog.Info(fmt.Sprintf("vocab size: %d", len(v.Tokens)))

	// add any additional tokens
	addIn, err := os.ReadFile(filepath.Join(dirpath, "added_tokens.json"))
	if os.IsNotExist(err) {
		return v, nil
	} else if err != nil {
		return nil, err
	}

	slog.Info("reading user defined tokens")

	var extraTokenData map[string]int
	if err := json.Unmarshal(addIn, &extraTokenData); err != nil {
		return nil, err
	}

	type token struct {
		key string
		pos int
	}

	extraTokens := make([]token, 0)
	for k, id := range extraTokenData {
		extraTokens = append(extraTokens, token{k, id})
	}

	slices.SortFunc(extraTokens, func(a, b token) int {
		return cmp.Compare(a.pos, b.pos)
	})

	numToks := len(v.Tokens)

	for cnt, t := range extraTokens {
		// the token id should match the specific index for the total number of tokens
		if t.pos != cnt+numToks {
			return nil, fmt.Errorf("token ID '%d' for '%s' doesn't match total token size", t.pos, t.key)
		}
		v.Tokens = append(v.Tokens, t.key)
		v.Scores = append(v.Scores, -1000.0)
		v.Types = append(v.Types, int32(llm.GGUFTokenUserDefined))
	}
	slog.Info(fmt.Sprintf("vocab size w/ extra tokens: %d", len(v.Tokens)))

	return v, nil
}

// Note: not all fields are modeled, only what's necessary for conversion
type AddedToken struct {
	ID         int32  `json:"id"`
	Content    string `json:"content"`
	SingleWord bool   `json:"single_word"`
	LStrip     bool   `json:"lstrip"`
	RStrip     bool   `json:"rstrip"`
	Normalized bool   `json:"normalized"`
	Special    bool   `json:"special"`
}

type TokenModel struct {
	Type   string           `json:"type"`
	Vocab  map[string]int32 `json:"vocab"`
	Merges []string         `json:"merges"`
}
type TokenizerJson struct {
	Version     string       `json:"version"`
	AddedTokens []AddedToken `json:"added_tokens"`
	Model       TokenModel   `json:"model"`
}

func loadJsonTokens(dirpath string) (*Vocab, error) {
	in, err := os.ReadFile(filepath.Join(dirpath, "tokenizer.json"))
	if err != nil {
		return nil, err
	}

	tokenizer := &TokenizerJson{}
	if err := json.Unmarshal(in, tokenizer); err != nil {
		return nil, err
	}

	v := &Vocab{
		Tokens: make([]string, len(tokenizer.Model.Vocab), len(tokenizer.Model.Vocab)+len(tokenizer.AddedTokens)),
		Scores: make([]float32, 0),
		Types:  make([]int32, len(tokenizer.Model.Vocab), len(tokenizer.Model.Vocab)+len(tokenizer.AddedTokens)),
		Merges: make([]string, len(tokenizer.Model.Merges)),
	}

	for tok, id := range tokenizer.Model.Vocab {
		if id < 0 || id-1 > int32(len(v.Tokens)) {
			return nil, fmt.Errorf("malformed tokens.json: id %d greater than length %d", id, len(v.Tokens))
		}
		v.Tokens[id] = tok
		v.Types[id] = TokenTypeNormal
	}
	for i, merge := range tokenizer.Model.Merges {
		v.Merges[i] = merge
	}

	for _, addedToken := range tokenizer.AddedTokens {
		if addedToken.ID >= int32(cap(v.Tokens)) {
			return nil, fmt.Errorf("malformed tokens.json: added id %d greater than length %d", addedToken.ID, cap(v.Tokens))
		}
		if addedToken.ID >= int32(len(v.Tokens)) {
			v.Tokens = v.Tokens[:int(addedToken.ID+1)]
			v.Types = v.Types[:int(addedToken.ID+1)]
		}
		v.Tokens[addedToken.ID] = addedToken.Content
		if addedToken.Special {
			v.Types[addedToken.ID] = TokenTypeControl
		} else {
			v.Types[addedToken.ID] = TokenTypeNormal
		}
	}

	slog.Info(fmt.Sprintf("vocab size: %d", len(v.Tokens)))
	slog.Debug("XXX loaded tokenizer.json", "addedTokens", tokenizer.AddedTokens, "vocabLen", len(tokenizer.Model.Vocab))
	return v, nil
}

func GetTensorName(n string) (string, error) {
	tMap := map[string]string{
		"model.embed_tokens.weight":                           "token_embd.weight",
		"model.layers.(\\d+).input_layernorm.weight":          "blk.$1.attn_norm.weight",
		"model.layers.(\\d+).mlp.down_proj.weight":            "blk.$1.ffn_down.weight",
		"model.layers.(\\d+).mlp.gate_proj.weight":            "blk.$1.ffn_gate.weight",
		"model.layers.(\\d+).mlp.up_proj.weight":              "blk.$1.ffn_up.weight",
		"model.layers.(\\d+).post_attention_layernorm.weight": "blk.$1.ffn_norm.weight",
		"model.layers.(\\d+).self_attn.k_proj.weight":         "blk.$1.attn_k.weight",
		"model.layers.(\\d+).self_attn.k_proj.bias":           "blk.$1.attn_k.bias",
		"model.layers.(\\d+).self_attn.o_proj.weight":         "blk.$1.attn_output.weight",
		"model.layers.(\\d+).self_attn.q_proj.weight":         "blk.$1.attn_q.weight",
		"model.layers.(\\d+).self_attn.q_proj.bias":           "blk.$1.attn_q.bias",
		"model.layers.(\\d+).self_attn.v_proj.weight":         "blk.$1.attn_v.weight",
		"model.layers.(\\d+).self_attn.v_proj.bias":           "blk.$1.attn_v.bias",
		"lm_head.weight":    "output.weight",
		"model.norm.weight": "output_norm.weight",
	}

	v, ok := tMap[n]
	if ok {
		return v, nil
	}

	// quick hack to rename the layers to gguf format
	for k, v := range tMap {
		re := regexp.MustCompile(k)
		newName := re.ReplaceAllString(n, v)
		if newName != n {
			return newName, nil
		}
	}

	return "", fmt.Errorf("couldn't find a layer name for '%s'", n)
}

func WriteGGUF(name string, tensors []llm.Tensor, params *Params, vocab *Vocab) (string, error) {
	c := llm.ContainerGGUF{
		ByteOrder: binary.LittleEndian,
	}

	var arch string
	switch len(params.Architectures) {
	case 0:
		return "", fmt.Errorf("No architecture specified to convert")
	case 1:
		switch params.Architectures[0] {
		case "MistralForCausalLM":
			arch = "llama"
		case "GemmaForCausalLM":
			arch = "gemma"
		case "Qwen2ForCausalLM":
			arch = "qwen2"
		default:
			return "", fmt.Errorf("Models based on '%s' are not yet supported", params.Architectures[0])
		}
	default:
		return "", fmt.Errorf("Multimodal models are not yet supported")
	}

	m := llm.NewGGUFModel(&c)
	m.Tensors = tensors

	m.KV["general.architecture"] = arch
	m.KV["general.name"] = name

	switch arch {
	case "llama":
		m.KV["llama.context_length"] = uint32(params.ContextSize)
		m.KV["llama.embedding_length"] = uint32(params.HiddenSize)
		m.KV["llama.block_count"] = uint32(params.HiddenLayers)
		m.KV["llama.feed_forward_length"] = uint32(params.IntermediateSize)
		m.KV["llama.rope.dimension_count"] = uint32(params.HiddenSize / params.AttentionHeads)
		slog.Debug(fmt.Sprintf("rope dim count = %d", m.KV["llama.rope.dimension_count"]))
		m.KV["llama.attention.head_count"] = uint32(params.AttentionHeads)
		m.KV["llama.attention.head_count_kv"] = uint32(params.KeyValHeads)
		m.KV["llama.attention.layer_norm_rms_epsilon"] = float32(params.NormEPS)
		m.KV["llama.rope.freq_base"] = float32(params.RopeFreqBase)
	case "gemma":
		m.KV["gemma.context_length"] = uint32(params.ContextSize)
		m.KV["gemma.embedding_length"] = uint32(params.HiddenSize)
		m.KV["gemma.block_count"] = uint32(params.HiddenLayers)
		m.KV["gemma.feed_forward_length"] = uint32(params.IntermediateSize)
		m.KV["gemma.attention.head_count"] = uint32(params.AttentionHeads)
		m.KV["gemma.attention.head_count_kv"] = uint32(params.KeyValHeads)
		m.KV["gemma.attention.layer_norm_rms_epsilon"] = float32(params.NormEPS)
		m.KV["gemma.attention.key_length"] = uint32(params.HeadDimension)
		m.KV["gemma.attention.value_length"] = uint32(params.HeadDimension)
	case "qwen2":
		m.KV["qwen2.context_length"] = uint32(params.ContextSize)
		m.KV["qwen2.embedding_length"] = uint32(params.HiddenSize)
		m.KV["qwen2.block_count"] = uint32(params.HiddenLayers)
		m.KV["qwen2.feed_forward_length"] = uint32(params.IntermediateSize)
		m.KV["qwen2.attention.head_count"] = uint32(params.AttentionHeads)
		m.KV["qwen2.attention.head_count_kv"] = uint32(params.KeyValHeads)
		m.KV["qwen2.attention.layer_norm_rms_epsilon"] = float32(params.NormEPS)
		m.KV["qwen2.use_parallel_residual"] = params.UseParallelResidual
	}

	m.KV["general.file_type"] = uint32(1)
	m.KV["tokenizer.ggml.model"] = "llama"

	m.KV["tokenizer.ggml.tokens"] = vocab.Tokens
	m.KV["tokenizer.ggml.scores"] = vocab.Scores
	m.KV["tokenizer.ggml.token_type"] = vocab.Types

	m.KV["tokenizer.ggml.bos_token_id"] = uint32(params.BoSTokenID)
	m.KV["tokenizer.ggml.eos_token_id"] = uint32(params.EoSTokenID)

	switch arch {
	case "llama":
		m.KV["tokenizer.ggml.unknown_token_id"] = uint32(0)
	case "gemma":
		m.KV["tokenizer.ggml.padding_token_id"] = uint32(params.PaddingTokenID)
		m.KV["tokenizer.ggml.unknown_token_id"] = uint32(3)
	}

	m.KV["tokenizer.ggml.add_bos_token"] = true
	m.KV["tokenizer.ggml.add_eos_token"] = false

	// llamacpp sets the chat template, however we don't need to set it since we pass it in through a layer
	// m.KV["tokenizer.chat_template"] = "{{ bos_token }}{% for message in messages %}{% if (message['role'] == 'user') != (loop.index0 % 2 == 0) %}{{ raise_exception('Conversation roles must alternate user/assistant/user/assistant/...') }}{% endif %}{% if message['role'] == 'user' %}{{ '[INST] ' + message['content'] + ' [/INST]' }}{% elif message['role'] == 'assistant' %}{{ message['content'] + eos_token}}{% else %}{{ raise_exception('Only user and assistant roles are supported!') }}{% endif %}{% endfor %}" // XXX removeme

	c.V3.NumTensor = uint64(len(tensors))
	c.V3.NumKV = uint64(len(m.KV))

	f, err := os.CreateTemp("", "ollama-gguf")
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = m.Encode(f)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}
