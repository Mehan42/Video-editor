# PageVideo — Отчёт о выполненной работе

**Дата:** 2026-08-06 — 2026-08-08
**Статус:** T1–T5 закрыты; кэш, study/faq/glossary, URL-ingress live-verified.

---

## Что сделано за сегодня (2026-08-06)

### 1. AGENTS.md — инструкции для AI-агентов
- Создан `AGENTS.md` на русском языке с компактными инструкциями для будущих сессий
- Содержит: команды запуска, особенности окружения (repo-local кэши), границы пакетов, нерушимые ограничения безопасности
- Указаны известные блокеры (race-тесты без GCC, отсутствие govulncheck)

### 2. Bionic (LM Studio) верифицирован и подключён
- Подтверждён запуск Bionic: `E:\LM Studio Bionic\Bionic.exe` слушает `127.0.0.1:1234`
- `provider check` возвращает **READY** с 6 локальными моделями:
  - `qwen/qwen3-vl-8b`, `llama-3.2-3b-instruct`, `qwen/qwen3.6-27b`
  - `google/gemma-4-12b-qat`, `prism-ml/bonsai-27b`, `text-embedding-nomic-embed-text-v1.5`
- Capability: `chat/completions`, `models`
- Readiness-запрос не содержит данных транскрипта

### 3. Безопасность LLM-границы усилена (internal/llm)
- Добавлены 7 новых httptest-тестов: malformed JSON, empty data, HTTP error, response-size limit, timeout, empty choices, malformed chat response
- Типизированные роли сообщений (`system|user|assistant`) с валидацией при encode
- `NewUserMessage` — только user-роль для недоверенного контента (транскрипт)
- `TestChatRequestConfigIsolation` — инъекция в транскрипте не может изменить модель/endpoint/capability
- `TestRejectsUnknownMessageRole` — неизвестные роли отклоняются
- Chat egress по-прежнему заблокирован по умолчанию (`ErrEgressBlocked`)

### 4. Opt-in Summary через локальный Bionic (internal/llm, config, pipeline, cli)
- Новый флаг `process --enable-summary` + `--llm-base-url`, `--llm-timeout`, `--llm-max-response-bytes`, `--summary-max-chars`
- `SummarizeTranscript` строит policy-separated запрос: system policy (в коде), task (в коде), transcript как user-message
- Модель определяется из readiness, не из контента
- `summary.md` записывается с SHA-256 хешем в `manifest.json`
- При ошибке LLM pipeline не падает — деградирует до READY без summary
- 5 новых тестов в `summarize_test.go`
- Live smoke на `smoke.mp4` прошёл успешно

### 5. Документация обновлена
- `CONTEXT.md` — отражён факт Bionic READY
- `docs/IMPLEMENTATION_STATUS.md` — расширен раздел llm, добавлены новые тесты и Bionic-статус
- `docs/NEXT_DAY_PLAN.md` — пункты 1–5 помечены DONE, добавлен раздел оставшегося scope
- `docs/roadmap.md` — исходная спецификация перенесена из корня в docs
- `docs/security/PROMPT_INJECTION_BOUNDARY.md` — зафиксирована policy-сепарация

### 6. CodeGraph обновлён
- Индекс обновлён после всех изменений исходников

### 7. Git: первый коммит и push
- Remote настроен: `github.com/Mehan42/Video-editor`
- Серия коммитов запушена (см. историю ниже)

---

## Что сделано за сегодня (2026-08-07)

### 1. UX-фиксы для CLI и REPL
- **Bare path/URL как input:** если первый аргумент — существующий локальный файл или `http(s)://` URL, CLI автоматически вызывает `process` с этим input (больше не нужно типа `process --input`)
- **Понятная ошибка для URL:** `URL input (…) is not supported yet: remote downloaders (YouTube/VK/RuTube/http) are not implemented`
- **REPL trim:** лидирующие пробелы в интерактивном режиме обрезаются
- **`--help` команда:** добавлена полная справка с описанием всех опций

### 2. Интерактивный REPL в launcher-е
- `scripts\pagevideo-start.bat` без аргументов открывает промпт `pagevideo>`
- Можно вводить команды одну за другой (`version`, `provider check`, `process --input …`)
- `exit`/`quit`/пустая строка — завершение сессии
- Двойной клик по батнику теперь не падает с ошибкой, а открывает справку

### 3. Стабильный бинарник и PATH
- Бинарник собран в `bin\pagevideo.exe` (актуальный билд с `--enable-summary`)
- Добавлен в PATH пользователя: `E:\Soft\PageVideo\bin`
- Создан `bin\pagevideo-start.bat` — ищет exe рядом с собой

### 4. Полный смок-сьют
Все проверки пройдены:

| Проверка | Результат |
|---|---|
| Unit-тесты (chunk, cli, llm) | PASS (~40 тестов, httptest only, без сети) |
| gofmt, go vet, build | OK |
| `git diff --check` | OK |
| CLI help/version | OK |
| REPL (интерактивный режим) | OK |
| Bare path / URL как input | OK |
| Provider check (live Bionic) | READY, 7 моделей |
| E2E на реальном видео (57 MB) | READY, transcript.txt 38 KB, хеши в manifest |
| Summary через Bionic (opt-in) | `summary.md` сгенерирован, хеш записан |
| Error paths | корректные сообщения |
| CodeGraph | up to date |

### 5. `.gitignore` дополнен
- `bin/` исключён из версионирования (бинарники воспроизводимы из исходников)

---

## Что сделано за сегодня (2026-08-08)

### 1. T2 — Run cache (`internal/cache`, `--no-cache`)
- Ключ: `sha256(input) + ParamsHash(ffmpeg-hash, whisper-hash, model-hash, language, chunk chars, overlap words)`.
- Хранение: `OutputRoot/.cache/<inputHash16>-<paramsHash16>/` содержит `audio.wav`, `transcript.txt`, `transcript.srt`, `cache.json`.
- Hit: файлы копируются в новый run-каталог, chunks пересобираются детерминированно, manifest переписывается с новым run_id. `manifest.json` и `cache.json` из кэша не копируются никогда.
- Любая ошибка в кэше (нет файла, hash drift, parse fail) трактуется как miss, не как падение.
- SaveCacheEntry best-effort: ошибка логируется, run не отменяется.
- Live-verified на `smoke.mp4`: hit-прогон даёт побайтово те же transcript/audio с новым run_id, лог `pagevideo: cache hit run=... source=...`.
- 6 unit-тестов в `internal/cache/cache_test.go`.

### 2. T3 — study/faq/glossary артефакты за `--enable-summary`
- `internal/llm/summarize.go` переименован `git mv` в `internal/llm/artifacts.go` и обобщён.
- Общий `sharedPolicy` (system) + per-artifact task text (`summaryTask`, `studyTask`, `faqTask`, `glossaryTask`). Транскрипт всегда как `user` через `NewUserMessage`.
- Новые методы: `GenerateStudy`, `GenerateFAQ`, `GenerateGlossary`; `SummarizeTranscript` стал обёрткой над общим `GenerateArtifact`.
- Размер ограничен `defaultArtifactMaxChars = 24000` байт единый для всех генераторов.
- Pipeline: `maybeSummarize` → `maybeArtifacts`. Результат: `Summary`, `Study`, `FAQ`, `Glossary` в `Result`; per-artifact failure logged, run не падает.
- Кэш *сознательно* не сохраняет LLM-артефакты — они недетерминированы.
- 4 новых httptest теста (`TestGenerateStudy_PolicySeparatedRequest`, `TestGenerateFAQ_UsesTaskSpecificHeader`, `TestGenerateGlossary_RejectsEmptyTranscript`, `TestGenerateArtifact_RejectsOversizedTranscript`).

### 3. T5 — Документация
- `docs/prompts.md`: весь system/task текст из `artifacts.go`. Изменение промпта требует правки и Go, и doc, и `IMPLEMENTATION_STATUS.md`.
- `docs/adr/ADR-008-stack-choices.md`: почему Go+ffmpeg+whisper.cpp+Markdown.
- `docs/adr/ADR-009-run-cache-key-strategy.md`: выбор ключа кэша, решение НЕ кэшировать LLM-артефакты.

### 4. T4 — URL-ingress через yt-dlp (`--allow-download`)
- Новый пакет `internal/download`: `Fetch(ctx, ytdlpPath, ffmpegPath, url, dstDir, maxBytes)`.
- `exec.CommandContext` с раздельными аргументами, никогда строкой.
- Флаги yt-dlp: `--no-call-home --no-playlist --no-warnings --quiet --ffmpeg-location <dir-of-ffmpeg> -f bv*+ba/b --merge-output-format mp4 --no-part -o download.%(ext)s`.
- После успешного выхода yt-dlp сканируется staging dir; должен быть ровно один `download.*` (иначе отказ).
- Валидация: regular file, внутри dstDir (no symlinks), `size > 0`, `size <= max-input-bytes`; oversized удаляется до возврата ошибки.
- Gate: `Config.Validate` требует `--allow-download` + наличие yt-dlp бинарника.
- `config.AbsolutePaths` больше не mangle-ит `https://...` в pseudo-path.
- `Result.SourceURL` записывается в manifest; **cache skip** при URL-входе (remote bytes могли измениться).
- Live-verified: `https://www.youtube.com/watch?v=jNQXAC9IVRw` (Me at the zoo, 19s) → transcript "Alright, so here we are, one of the elephants..." под `.pagevideo/url-smoke/<run>/`. 6 unit-тестов без сети.

### 5. CodeGraph
- `codegraph sync` выполнен: 15 файлов, 262 узла.

---

## История коммитов (main, github.com/Mehan42/Video-editor)

```
c7a50f7 feat(download): opt-in URL input via yt-dlp (--allow-download)
dc35e63 docs: capture prompts as data + ADRs for stack and run-cache strategy
64b7860 feat(llm): study/faq/glossary artifacts under --enable-summary
70dbf30 feat(cache): local run cache for repeated process invocations
9ec8167 docs: record 2026-08-07 UX fixes, full smoke suite, Bionic 7-model READY state
8061863 chore: add bin/ to gitignore
103c4c5 feat(cli): accept bare path or URL as input, clearer URL error
78f74a3 feat(cli): interactive REPL in launcher when run without args
fe8d715 feat(llm): add opt-in summary via local Bionic chat
5bda209 docs: mark next-day plan items done, list remaining deferred scope
a7c556d docs(status): record llm test coverage and prompt-boundary guarantees
557eaa5 security(llm): enforce prompt-injection and authority boundary in code
bfab4b3 test(llm): extend httptest fixtures for provider contract
1c824a9 docs: stage AGENTS.md, move roadmap to docs, record Bionic READY state
```

---

## Что НЕ сделано (осознанно отложено)

- Eviction policy для `.cache/` и `.staging/` (cache prune CLI, mtime/size trim)
- Кэширование LLM-артефактов (summary/study/faq/glossary) — см. ADR-009
- Дополнение `docs/security/MEDIA_SANDBOX.md` рисками downloader-ов (SSRF, redirect, content-type)
- Resume после mid-stage fail (WAL, чекпоинты стадий)
- Knowledge-store index: связь run_id → topics across runs
- Выборочный список артефактов (напр. `--artifacts study,faq` вместо всех четырёх summary-флагом)
- quiz/checklist/flashcards/Mermaid-генерация
- NATS transport, MCP connectors, provider registry
- OS-level media sandbox и job-object limits
- Автономный агент и оркестрация
- Obsidian/RAG экспорт
- `go test -race` (нет GCC/cgo), `govulncheck` (не установлен)

---

## Как использовать сейчас

```powershell
cd E:\Soft\PageVideo

# Интерактивный режим (двойной клик или без аргументов)
scripts\pagevideo-start.bat

# Полный pipeline с summary+study+faq+glossary
scripts\pagevideo-start.bat process --input "D:\Видео\урок.mp4" --enable-summary

# URL через yt-dlp (opt-in egress)
scripts\pagevideo-start.bat process --input "https://www.youtube.com/watch?v=..." --allow-download

# Без кэша (полный recompute)
scripts\pagevideo-start.bat process --input "D:\Видео\урок.mp4" --no-cache

# Проверка Bionic
scripts\pagevideo-start.bat provider check
```

Результаты в `output\<run-id>\`: `transcript.txt`, `transcript.srt`, при `--enable-summary` также `summary.md`, `study.md`, `faq.md`, `glossary.md`, плюс `manifest.json` с хешами. Кэш — в `output\.cache\`; staging URL-скачиваний — в `output\.staging\`.
