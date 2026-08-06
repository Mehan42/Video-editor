# Local Run

## Launcher

Run the application through:

```bat
scripts\pagevideo-start.bat process --input "path\to\video.mp4"
```

The launcher resolves the project root from its own location, builds `.pagevideo\pagevideo.exe` if it is missing, and forwards all arguments to the CLI. It does not start Bionic, NATS, MCP, external providers, or network services automatically.

## Transcription example

```bat
scripts\pagevideo-start.bat process --input "D:\Media\lesson.mp4" --output "D:\Media\pagevideo-output"
```

The local dependencies must exist at `ffmpeg\bin\ffmpeg.exe`, `whisper.cpp\bin\whisper-cli.exe`, and `whisper.cpp\models\ggml-base.bin`, or be supplied with explicit CLI flags.

## Provider readiness

After Bionic is started manually and its local server is enabled, check readiness without sending transcript data:

```bat
scripts\pagevideo-start.bat provider check --base-url "http://127.0.0.1:1234/v1"
```

`READY` requires a successful `/v1/models` response. `BLOCKED_PROVIDER` is the truthful result when no local listener is running. Chat remains blocked by policy unless explicitly enabled in code/configuration.

## Safety boundary

The launcher executes only the local PageVideo binary. It does not interpret video metadata as commands, launch arbitrary tools, enable network access, or publish generated artifacts.
