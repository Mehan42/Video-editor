# ADR-008: Принятие стека — Go, ffmpeg, whisper.cpp, Markdown

**Status:** accepted (фиксация давно действующей практики)  
**Date:** 2026-08-08  
**Deciders:** operator + ai-agent (T5 docs sync)

## Контекст

Roadmap (перенесён в `docs/roadmap.md` из исходного брифа) постулирует набор технологий. Этот ADR фиксирует *почему* именно они, чтобы через год не пришлось реverse-engineer мотивацию.

## Решения

### Go (stdlib only)

- **Почему:** один статический `.exe` на Windows, нет runtime-установки, кросс-компиляция тавтологична. Сильный `os/exec`, `crypto/sha256`, `encoding/json` покрывают весь наш пайплайн: оркестрация процессов, хеширование, атомарная запись manifest. Строгая типизация материализует границы безопасности (`Config`, `Artifact`, закрытый enum ролей).
- **Отказ:** Python+click и Node.js дают более богатые LLM-SDK, но требуют venv/npm окружения и гораздо труднее втащить в portable distro.
- **Цена:** работа с LLM идёт через raw HTTP + `encoding/json`; нет стриминга. Принято — наш обсяг (chat, /v1/models) тривиален.
- **Ограничение:** только stdlib. Любая внешняя зависимость — отдельный ADR.

### ffmpeg (внешний бинарник, не cgo)

- **Почему:** нулевая статическая линковка, независимое обновление, поддержка всего зоопарка контейнеров (mp4/mkv/mov/avi), hwaccel без переписывания. Запускаем через `exec.CommandContext` с раздельными аргументами — нет shell-string, нет инъекций.
- **Отказ:** CGO-обёртки (`go-ffmpeg`, `gmf`) тащат C-стек и ломают "только stdlib".
- **Цена:** зависим от внешнего `\ffmpeg\bin\ffmpeg.exe`; его SHA-256 входит в ключ кэша (см. ADR-009) — подмена бинарника инвалидирует кэш.

### whisper.cpp (`whisper-cli` binary, не daemon)

- **Почему:** локально, без egress; модель `ggml-*.bin` — честный файл на диске (хешируется в ключ кэша). Субпроцесс-модель = удобный sandbox boundary: whisper не имеет доступа к нашей памяти.
- **Отказ:** faster-whisper (Python), OpenAI Whisper API (не соответствует local-first постулату).
- **Цена:** стоимость первой загрузки модели при каждом run; сглаживается кэшем (T2).

### Markdown + JSON

- **Почему:** LLM умеет Markdown нативно и стабильно; Obsidian/VS Code/Notion/Telegram принимают без конвертеров. `manifest.json` — единый точка-входа машино-читаемого состояния (хеши всех артефактов, run id, chunk IDs).
- **Отказ:** PDF/DOCX требуют конвертера; YAML human-friendly, но stdlib его не парсит.
- **Цена:** никакой подсветки синтаксиса, изображений и embed-ов; принято.

## Последствия

- Весь проект — **один** бинарник + два внешних исполняемых + одна GGML-модель. Полный runtime < 200 МБ.
- Под Windows работаем без WSL/Docker/VM.
- Существующие ADR (005–007, untrusted-границы) продолжают действовать; этот ADR их не отменяет.

## Ссылки

- `docs/roadmap.md` (источник брифа)
- ADR-005 (untrusted media isolation), ADR-006 (prompt injection), ADR-007 (secret/egress boundary)
- `internal/pipeline/pipeline.go` (`exec.CommandContext` с раздельными аргументами)
