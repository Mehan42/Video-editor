# ADR-009: Run-cache keyed by input hash plus dependency hash

**Status:** accepted
**Date:** 2026-08-08
**Deciders:** operator + ai-agent (T2)

## Контекст

Прогон whisper на 30-минутном видео занимает минуты на CPU-only. Редактирование выходных артефактов (разные чанк-параметры, включение summary) итерируется без изменения видео — повторять ffmpeg+whisper бессмысленно.

Возможные подходы:

1. **Ничего не кэшировать** (как было до T2) — просто, но больно на практике.
2. **Полный WAL/resume** — вести журнал стадий, уметь продолжать с последнего чекпоинта. Гибко, но надо проектировать под все внутренние точки отказа.
3. **Read-only reuse** — на hit: копировать готовые `audio.wav`+`transcript.txt`+`transcript.srt` в новый run-каталог, переписывать manifest; ffmpeg/whisper не запускаем.

Выбран 3-й.

## Решение

- Ключ: `sha256(input)` ⊕ `ParamsHash(sha256(ffmpeg), sha256(whisper), sha256(model), language, chunk_chars, overlap_words)`.
- Локация: `OutputRoot/.cache/<inputHash16>-<paramsHash16>/`.
- Храним: `audio.wav`, `transcript.txt`, `transcript.srt`, `cache.json` (метаданные + хеши файлов).
- При hit: копируем файлы в новый run-dir, **пересобираем chunks детерминированно** (новый runID в telemetry), переписываем manifest с новыми путями — `manifest.json` и `cache.json` НЕ копируются из кэша.
- Инвалидация: любое изменение входа, ffmpeg/whisper/model байт, языка, chunk геометрии → новый ключ → miss.
- Отказоустойчивость: любой check-fail в `cache.Load` (нет файла, hash drift, JSON parse) → miss + full recompute, никогда не падение. Сохранение кэша — best-effort, ошибка не роняет run.
- `--no-cache` для явного bypass.

## Последствия

- ⚠️ Cache не включает `summary.md`/`study.md`/`faq.md`/`glossary.md` — они недетерминированы (LLM); на каждом run с `--enable-summary` LLM вызывается заново. Если LLM-артефакты тоже нужно кэшировать, это отдельное решение (и отдельный ADR).
- ⚠️ disk: ~2 × audio+transcript на уникальный (input, params). Приемлемо для локального использования; eviction policy отсутствует — оператор удаляет старые `.cache/<key>` вручную.

## Альтернативы

- Сохранять только transcript, а audio не хранить — экономит ~55MB на раn, но усложняет `Result` (audio artifact ссылается на cache path вне run dir). Отвергнуто как преждевременная оптимизация.
- GC по LRU — добавит зависимость от mtime. Отложено до реального давления на диск.

## Ссылки

- `internal/cache/cache.go`, `internal/cache/cache_test.go`
- `docs/IMPLEMENTATION_STATUS.md` (2026-08-08 секция)
