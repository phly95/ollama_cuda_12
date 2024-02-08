# Ollama Windows Preview

Welcome to the Ollama Windows preview.

No more WSL required!

Ollama now runs as a native Windows application, with full access to your GPU
(NVIDIA only at this time.)  Ollama does not require Admin privileges.  Ollama
on Windows consists of a GUI tray app, the CLI client which can be run in `cmd`,
`powershell` or your favorite terminal app, and the server process.  This server
binds to `localhost:11434` and exposes the Ollama REST API and can be used by
other applications to run LLMs.
  
As this is a preview release, you should expect a few bugs here and there. We're
tracking these in a single issue [INSERT LINK HERE] for the duration of the
preview.  If you run into a problem, please take a look at that issue. If
someone already reported the same issue, please +1 their comment so we can see
how common it is.  If your problem looks unique, please add a new comment and
attach logs (see [Troubleshooting](#troubleshooting) below)

## System Requirements

* Windows 10 or newer, Home or Pro
* NVIDIA Drivers if you have an NVIDIA card

TODO - figure out minimum driver version compatible with cuda v11.3

## API Access

Here's a quick example showing API access from `powershell`
```powershell
(Invoke-WebRequest -method POST -Body '{"model":"llama2", "prompt":"Why is the sky blue?", "stream": false}' -uri http://localhost:11434/api/generate ).Content | ConvertFrom-json
```

## Troubleshooting

While we're in preview, `OLLAMA_DEBUG` is always enabled, which adds
a "view logs" menu item to the app, and increses logging for the GUI app and
server.

Ollama on Windows stores files in a few different locations.  You can view them in
the explorer window by hitting `<cmd>+R` and type in:
- `explorer %LOCALAPPDATA%\Ollama` contains logs, and downloaded updates
    - *app.log* contains logs from the GUI application
    - *server.log* contains the server logs
    - *upgrade.log* contains log output for upgrades
- `explorer %LOCALAPPDATA%\Programs\Ollama` contains the binaries (The installer adds this to your user PATH)
- `explorer %HOMEPATH%\.ollama` contains models and configuration
- `explorer %TEMP%` contains temporary executable files in one or more `ollama*` directories
