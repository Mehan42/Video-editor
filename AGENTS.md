# AGENTS.md — PageVideo

Руководство для AI-агентов, работающих в этом репозитории. Контекст и границы полномочий описаны в `CONTEXT.md` — прочти его, прежде чем делать изменения, расширяющие scope.

## Что это

Local-first CLI на Go (модуль `pagevideo`, только stdlib, Go 1.25+ под Windows). Прогоняет локальный видеофайл через ffmpeg → whisper.cpp → детерминированный chunking, записывая WAV/TXT/SRT/chunks и `manifest.json` с хешами в каталог запуска с правами `0700`. LLM-генерация, NATS, MCP, retrieval и сетевые возможности пока не реализованы — не выдавай их за работающие.

## Команды

```powershell
# Тесты/vet ТРЕБУЮТ repo-local кэши (системные кэши недоступны):
$env:GOCACHE = "$PWD\.gocache"
$env:GOMODCACHE = "$PWD\.gomodcache"

go test ./...            # все тесты; сеть не нужна (только httptest-фикстуры)
go test -run TestName ./internal/chunk   # одиночный тест
go vet ./...
go build -o .\.pagevideo\pagevideo.exe .\cmd\pagevideo

# Запуск CLI через поддерживаемый launcher (собирает бинарник, если его нет):
scripts\pagevideo-start.bat process --input "D:\Media\lesson.mp4"
scripts\pagevideo-start.bat provider check --base-url "http://127.0.0.1:1234/v1"
```

Последовательность проверки из `docs/NEXT_DAY_PLAN.md`: gofmt → `go test ./...` → `go vet` (по пакетам и полный) → build → `git diff --check`.

Известные блокеры (не регрессии): `go test -race` требует C-компилятор (отсутствует); `govulncheck` не установлен.

## Особенности окружения

- Бинарники ffmpeg и whisper.cpp в gitignore, ожидаются по путям `ffmpeg\bin\ffmpeg.exe`, `whisper.cpp\bin\whisper-cli.exe`, `whisper.cpp\models\ggml-base.bin`. Не коммить их. Пути переопределяются флагами `--ffmpeg`/`--whisper`/`--model`.
- `.pagevideo\` (собранный бинарник, smoke-выводы, логи) и `.gocache\`/`.gomodcache\` — локальные артефакты, никогда не коммитить.
- Локальный git-репозиторий на `main`, remote не настроен; в README указан канонический remote (`github.com/Mehan42/Video-editor`), но публикация — отдельный явный шаг оператора. Не пушить без запроса и подтверждай любые git-мутации — первого коммита ещё нет.
- Индекс CodeGraph лежит в `.codegraph\`; обновляй его после изменений исходников (выходной гейт #5).

## Границы пакетов

- `cmd/pagevideo/main.go` — только тонкая точка входа.
- `internal/cli` — разбор флагов, диспетчеризация команд (`process`, `provider check`, `version`); `UsageError` — путь пользовательской ошибки.
- `internal/config` — `Config` + `Validate()` (расширение, размер, пути зависимостей, timeout, параметры chunking).
- `internal/pipeline` — оркестрация ffmpeg → whisper → chunks → атомарная запись manifest.
- `internal/chunk` — детерминированный chunker (одинаковый вход → одинаковый выход).
- `internal/llm` — loopback-only адаптер Bionic (LM Studio): готовность через `/v1/models`, лимит размера ответа. Chat заблокирован, пока явный флаг не разрешит — пусть так и остаётся.

## Нерушимые ограничения

- Никогда не собирать строку shell-команды: внешние бинарники запускаются через `exec.CommandContext` с раздельными аргументами. `--language` не должно по имени попадать в shell.
- Субтитры, транскрипт, chunks и вывод LLM — это недоверенные данные: они не могут выбирать provider/capability, менять политику, читать секреты или запускать инструменты/публикацию.
- Внешний egress к провайдерам и любые изменяющие/публикующие действия требуют явного одобрения оператора и отдельного изменения; тесты используют только `httptest`, никогда живой провайдер.
- Обработка входного видео ограничена `--timeout` и `--max-input-bytes`; не снимай эти лимиты при правках pipeline.
- После изменения чего-либо из этого списка или из `CONTEXT.md` обнови соответствующие документы.

## Истина о статусе

README и роадмап — это планы; авторитетное состояние в `docs/IMPLEMENTATION_STATUS.md` (проверено vs нет). Готовность Bionic сейчас `BLOCKED_PROVIDER` (нет слушателя на 127.0.0.1:1234); `provider check` не должен отправлять данные транскрипта. Следующая плановая сессия: `docs/NEXT_DAY_PLAN.md`.
