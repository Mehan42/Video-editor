# PageVideo

PageVideo — локальный CLI-пайплайн, который превращает видео в проверяемый текстовый материал: извлекает аудио, выполняет транскрибацию, сохраняет TXT/SRT, разбивает транскрипт на детерминированные chunks и пишет manifest с hash/provenance.

Проект развивается как local-first Knowledge Extraction Pipeline. Целевая идея из [docs/roadmap.md](docs/roadmap.md):

```text
Видео → данные → текст → структура → знания → база знаний → обучение
```

Текущий код реализует безопасный первый срез этой цепочки. LLM-генерация, RAG, NATS/MCP и внешняя публикация пока не считаются готовыми возможностями.

## Что уже работает

- локальный CLI на Go;
- входные MP4, MOV, MKV и AVI;
- извлечение mono WAV через локальный ffmpeg;
- транскрибация через локальный whisper.cpp;
- `transcript.txt` и `transcript.srt`;
- детерминированная разбивка текста на chunks с overlap;
- SHA-256 для входа и производных артефактов;
- `manifest.json` с run ID, hash, provenance и trust class;
- timeout и лимит размера входного файла;
- вызов внешних бинарников без shell-команд;
- локальный readiness-check для LM Studio Bionic;
- Windows launcher `scripts\pagevideo-start.bat`.

## Что пока не реализовано

Из целей и задач исходной спецификации пока остаются в работе:

- YouTube, VK и RuTube downloader adapters;
- определение глав и semantic chunking;
- генерация `summary.md`, `study.md`, `faq.md`, `glossary.md`, `quiz.md` и `checklist.md`;
- кэширование и возобновление повторных запусков;
- LLM Gateway с реальным chat generation;
- NATS transport и MCP connectors;
- OpenRouter aliases, FreeLLM и другие provider adapters;
- Retriever, embeddings, Obsidian export и полноценный RAG;
- автономный агент;
- OS-level media sandbox и job/resource limits;
- автоматическая публикация или изменение внешней базы знаний.

## Требования

- Windows;
- Go 1.25 или новее;
- локальные зависимости в рабочем каталоге:

```text
ffmpeg\bin\ffmpeg.exe
whisper.cpp\bin\whisper-cli.exe
whisper.cpp\models\ggml-base.bin
```

Пути можно заменить флагами CLI `--ffmpeg`, `--whisper` и `--model`.

## Быстрый запуск

Рекомендуемый Windows-путь:

```bat
scripts\pagevideo-start.bat process --input "D:\Media\lesson.mp4"
```

Launcher разрешает корень проекта относительно своего расположения, собирает `.pagevideo\pagevideo.exe`, если бинарник отсутствует, и передаёт аргументы CLI.

Эквивалентный прямой запуск:

```powershell
go run .\cmd\pagevideo process `
  --input "D:\Media\lesson.mp4" `
  --output "D:\Media\pagevideo-output"
```

Команда создаёт отдельный каталог запуска и возвращает JSON с путями к WAV, TXT, SRT, chunks и manifest.

## Проверка Bionic

Локальная установка LM Studio Bionic обнаружена по пути `E:\LM Studio Bionic\Bionic.exe`. Кандидатный OpenAI-compatible endpoint — `http://127.0.0.1:1234/v1`, но readiness зависит от вручную запущенного локального сервера.

Проверка моделей не отправляет транскрипт:

```bat
scripts\pagevideo-start.bat provider check --base-url "http://127.0.0.1:1234/v1"
```

Результаты:

- `READY` — `/v1/models` ответил успешно;
- `BLOCKED_PROVIDER` — локальный listener недоступен;
- chat egress заблокирован по умолчанию и не включается самим launcher.

## Безопасность

Видео, metadata, transcript, chunks, retrieved context и LLM output считаются untrusted data. Они не могут выбирать provider, включать capability, читать секреты или запускать инструменты.

В текущем MVP:

- ffmpeg и Whisper вызываются отдельными процессами;
- shell parsing не используется;
- вход и длительность обработки ограничены;
- output пишется в отдельный каталог с ограниченными правами;
- LLM chat заблокирован без явного флага;
- external egress и автоматическая публикация не выполняются.

Полная модель угроз и оставшиеся security gates описаны в [docs/security/THREAT_MODEL.md](docs/security/THREAT_MODEL.md), [docs/security/MEDIA_SANDBOX.md](docs/security/MEDIA_SANDBOX.md) и [docs/security/LLM_EGRESS_POLICY.md](docs/security/LLM_EGRESS_POLICY.md).

## Архитектура и документация

- [CONTEXT.md](CONTEXT.md) — текущее состояние и границы authority;
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — общая схема компонентов;
- [docs/architecture/PIPELINE.md](docs/architecture/PIPELINE.md) — стадии pipeline;
- [docs/architecture/LLM_GATEWAY.md](docs/architecture/LLM_GATEWAY.md) — provider-neutral LLM boundary;
- [docs/architecture/RAG.md](docs/architecture/RAG.md) — chunking, store и retrieval;
- [docs/IMPLEMENTATION_STATUS.md](docs/IMPLEMENTATION_STATUS.md) — проверенный статус;
- [docs/NEXT_DAY_PLAN.md](docs/NEXT_DAY_PLAN.md) — ближайший план разработки;
- [docs/LOCAL_RUN.md](docs/LOCAL_RUN.md) — локальный запуск и диагностика.

## Roadmap из исходной спецификации

- `v0.1` — CLI, ffmpeg, whisper.cpp, TXT, SRT;
- `v0.2` — Markdown, summary, chapters;
- `v0.3` — FAQ, glossary, checklist;
- `v0.4` — quiz, flashcards, Mermaid;
- `v0.5` — OpenAI-compatible providers, OpenRouter, FreeLLM, Bionic;
- `v1.0` — Knowledge Base, Obsidian и RAG.

Roadmap не является утверждением о готовности функций: фактический статус определяется кодом, тестами и evidence в `docs/IMPLEMENTATION_STATUS.md`.

## Проверка разработки

```powershell
$env:GOCACHE = "$PWD\.gocache"
$env:GOMODCACHE = "$PWD\.gomodcache"
go test ./...
go vet ./internal/llm ./internal/cli ./internal/config ./internal/chunk ./internal/pipeline
go build -o .\.pagevideo\pagevideo.exe .\cmd\pagevideo
```

`go test -race ./...` пока требует доступный C compiler для cgo. `govulncheck` должен быть добавлен отдельным security gate.

## Репозиторий

Канонический remote: [github.com/Mehan42/Video-editor](https://github.com/Mehan42/Video-editor). Публикация выполняется только отдельной командой после проверки staged scope, тестов и локального Git-состояния.
