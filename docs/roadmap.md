# Исходная спецификация и roadmap

Исходный бриф проекта (исторический документ). Нормативный статус кода — в `docs/IMPLEMENTATION_STATUS.md`.

Да. Причем я бы сделал проект сразу как **production-ready CLI**, а не "скриптик на вечер". Тогда его можно будет расширять годами.

Я бы положил в корень примерно такую структуру:

```text
learnify/
│
├── cmd/
│   └── learnify/
│       └── main.go
│
├── internal/
│   ├── ffmpeg/
│   ├── whisper/
│   ├── llm/
│   ├── markdown/
│   ├── chunker/
│   ├── prompts/
│   ├── pipeline/
│   ├── config/
│   └── output/
│
├── docs/
│   ├── spec.md
│   ├── roadmap.md
│   ├── context.md
│   ├── architecture.md
│   └── prompts.md
│
├── examples/
├── models/
├── output/
├── README.md
└── go.mod
```

---

# README.md

Это "что это вообще такое".

Я бы включил:

```
# Learnify

CLI, превращающий любое видео в учебник.

Pipeline

Video
 ↓
ffmpeg
 ↓
whisper.cpp
 ↓
Transcript
 ↓
LLM
 ↓
Knowledge Base

Возможности

✔ локальная работа

✔ Go

✔ ffmpeg

✔ whisper.cpp

✔ Markdown

✔ Obsidian

✔ RAG-ready

✔ OpenAI/OpenRouter/Ollama
```

---

# spec.md

Самый главный документ.

Например

```
Цель

Преобразование видео в структурированную базу знаний.

Вход

mp4
mov
mkv
avi
YouTube URL
VK URL
RuTube URL

Выход

transcript.txt

transcript.srt

summary.md

study.md

faq.md

glossary.md

quiz.md

checklist.md

metadata.json
```

---

Дальше

```
Функциональные требования

✓ извлечение аудио

✓ распознавание

✓ разбивка по главам

✓ генерация Markdown

✓ кэширование

✓ повторный запуск

✓ работа офлайн

✓ плагины LLM
```

---

# roadmap.md

Я люблю делать по версиям.

```
v0.1

CLI

ffmpeg

whisper.cpp

txt

srt

----------------

v0.2

Markdown

Summary

Chapters

----------------

v0.3

FAQ

Glossary

Checklist

----------------

v0.4

Quiz

Flashcards

Mermaid

----------------

v0.5

OpenAI

OpenRouter

Ollama

----------------

v1.0

Knowledge Base

Obsidian

RAG
```

---

# context.md

Это документ, который потом будет использоваться LLM.

Например

```
Ты технический редактор.

Никогда не пересказывай буквально.

Используй терминологию автора.

Исправляй ошибки речи.

Удаляй слова-паразиты.

Не теряй смысл.

Если встречается инструкция —

оформляй как чек-лист.

Если встречается теория —

оформляй как учебник.

Если встречаются определения —

оформляй как глоссарий.

Если встречаются советы —

оформляй как Best Practices.
```

Это позволит менять поведение модели без изменения кода.

---

## Я бы добавил ещё несколько документов

### prompts.md

Все промпты отдельно.

```
summary

faq

study

quiz

flashcards

glossary

mindmap

checklist
```

Тогда менять качество генерации можно без перекомпиляции.

---

### architecture.md

```
Video

↓

Downloader

↓

ffmpeg

↓

Whisper

↓

Chunker

↓

Prompt Engine

↓

LLM

↓

Markdown Generator

↓

Output
```

---

### decisions.md (ADR)

Почему выбран именно такой стек.

```
Почему Go

Почему ffmpeg

Почему whisper.cpp

Почему Markdown

Почему Obsidian

Почему Mermaid

Почему JSON Metadata
```

Через год ты уже не будешь вспоминать, зачем принимал те или иные решения.

---

## Я бы ещё изменил концепцию проекта

Я бы не называл его просто **learnify**.

По сути это уже **Knowledge Extraction Pipeline**.

То есть это не "транскрибатор", а система извлечения знаний.

Получается примерно такая философия:

```
Видео
      ↓
Данные
      ↓
Текст
      ↓
Структура
      ↓
Знания
      ↓
База знаний
      ↓
Обучение
```

Именно поэтому я бы сделал `spec.md` главным документом проекта, а `roadmap.md` и `context.md` — вторыми по важности. Если архитектура и требования хорошо описаны с самого начала, код на Go потом пишется значительно проще и остаётся расширяемым.
