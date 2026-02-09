# Staticlint - Multichecker для статического анализа Go кода

## Описание

Staticlint - это multichecker, объединяющий множество статических анализаторов для Go.
Он предназначен для комплексной проверки кода на соответствие лучшим практикам,
обнаружения потенциальных ошибок и проблем с безопасностью.

## Состав анализаторов

### 1. Стандартные анализаторы (golang.org/x/tools/go/analysis/passes)

- **asmdecl** - проверка соответствия объявлений assembly
- **assign** - обнаружение бесполезных присваиваний
- **atomic** - проверка использования sync/atomic
- **bools** - обнаружение ошибок с булевыми операторами
- **buildssa** - построение SSA-формы
- **buildtag** - проверка тегов сборки
- **cgocall** - обнаружение нарушений в вызовах cgo
- **composite** - проверка литералов составных типов
- **copylock** - проверка копирования значений, содержащих мьютексы
- **ctrlflow** - анализ потока управления
- **deepequalerrors** - проверка использования reflect.DeepEqual с ошибками
- **errorsas** - проверка второго аргумента errors.As
- **fieldalignment** - выравнивание полей структур
- **findcall** - поиск вызовов функций
- **framepointer** - анализ указателей фреймов
- **httpresponse** - проверка закрытия HTTP-ответов
- **ifaceassert** - обнаружение невозможных утверждений типов интерфейса
- **inspect** - инспектор AST (базовый анализатор)
- **loopclosure** - проверка замыканий в циклах
- **lostcancel** - проверка отмены контекстов
- **nilfunc** - обнаружение сравнений с nil
- **nilness** - анализ nil-значений
- **pkgfact** - сбор фактов о пакетах
- **printf** - проверка форматных строк
- **reflect** - проверка использования reflect
- **shadow** - обнаружение затенения переменных
- **shift** - проверка сдвигов
- **sigchanyzer** - анализ каналов сигналов
- **sortslice** - проверка сортировки срезов
- **stdmethods** - проверка стандартных методов
- **stringintconv** - проверка преобразований строк↔числа
- **structtag** - проверка тегов структур
- **testinggoroutine** - обнаружение горутин в тестах
- **tests** - проверка тестов
- **timeformat** - проверка форматирования времени
- **unmarshal** - проверка разбора JSON/XML
- **unreachable** - обнаружение недостижимого кода
- **unsafeptr** - проверка использования unsafe.Pointer
- **unusedresult** - обнаружение неиспользуемых результатов
- **unusedwrite** - обнаружение неиспользуемых записей

### 2. Анализаторы staticcheck (класс SA - Стандартные)

Все анализаторы класса SA, включая:
- **SA1000** - Invalid regular expression
- **SA4000** - Binary operator has identical expressions
- **SA5000** - Assignment to nil map
- и другие...

### 3. Другие классы staticcheck

- **S1000** (Style) - Use plain channel send or receive
- **ST1000** (Style) - Incorrect or missing package comment
- **QF1001** (Quick Fix) - Apply De Morgan's law
- **S1001** (Simple) - Replace for loop with call to copy

### 4. Кастомный анализатор exitcheck

**exitcheck** - запрещает прямой вызов `os.Exit` в функции `main` пакета `main`.
Это позволяет defer-функциям корректно выполниться перед завершением программы.

## Установка и запуск

### Установка зависимостей

```bash
go mod init github.com/besapuz/urlshort
go get golang.org/x/tools/go/analysis/passes@latest
go get honnef.co/go/tools/staticcheck@latest