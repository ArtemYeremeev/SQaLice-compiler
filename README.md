# SQaLice-compiler

<a href="https://ibb.co/4YJPnGn"><img src="https://i.ibb.co/4YJPnGn/SQa-Lice-logo.png" alt="SQa-Lice-logo" border="0"></a>

## Описание компиляции запроса

- [Содержание](#содержание)
- [Описание и назначение](#описание-и-назначение)
- [Формат запроса](#формат-запроса)
- [Структура запроса](#структура-запроса)
- [Типы математических операторов](#типы-математических-операторов)
- [Типы логических операторов](#типы-логических-операторов)
- [Блок fields](#блок-fields)
- [Блок conditions](#блок-conditions)
- [Блок restrictions](#блок-restrictions)
- [TO-DO](#to-do)

## Описание методов парсинга запроса

- [Описание методов парсинга запроса](#описание-методов-парсинга-запросов)
- [Список полей](#список-полей)
- [Список условий](#список-условий)
- [Условие по названию](#условие-по-названию)
- [Поле сортировки](#поле-сортировки)
- [Порядок сортировки](#порядок-сортировки)
- [Лимит](#лимит)
- [Отступ](#отступ)

## Описание и назначение

Пакет компилятора SQaLice предназначен для гибкой обработки запросов к БД на PostgreSQL в проектах с применением go-swagger. SQaLice позволяет запрашивать конкретный набор полей из установленного __target__, фильтровать, сортировать и ограничивать получаемый набор строк.

## Подготовка к использованию компилятора

В функцию-компилятор *Compile* в качестве аргумента передается __modelsMap__, содержащая информацию о полях моделей проекта, используемых для обработки целевого SQL кода.
Рекомендуется формировать данную map динамически, например в init модуля при запуске проекта, что обеспечит консистентность сгенерированных посредством go-swagger
моделей и данных, передаваемых в компилятор через map. Для этого можно использовать функцию __FormDinamicModel__. Также в этом случае в swagger-модели не должно быть вложенных моделей, иначе произойдут некорректное считывание тегов полей и ошибки формирования полей.

## Формат запроса

В случае обращения к компилятору SQaLice все параметры целевого запроса должны содержаться в аргументе __params__. В __target__ передается целевая таблица или view, к которой осуществляется запрос и которая используется при указании динамической модели.

## Структура запроса

Аргумент __params__ при парсинге запроса внутри *Compile* разделяется на 3 блока - __fields__, __conditions__, __restrictions__.

### Пример адресной строки запроса, содержащей все 3 блока

```http
http://url/.../query=ID,title,updatedAt?ID==10||ID==8?ID,desc,2,0
```

В случае отсутствия необходимости в указании одного из этих блоков, приведенное форматирование должно сохраняться.

### Пример адресной строки запроса без блока conditions

```http
http://url/.../query=ID,title,updatedAt??ID,desc,2,0
```

Запрос без указания условий в 3 блоках одновременно не рекомендуется, так как может привести к неожиданным ошибкам.

## Типы математических операторов

SQaLice поддерживает следующие математические операторы:

| Оператор         | SQaLice | PG    |
| ---------------- | ------- | ----- |
| РАВНО            | ==      | =     |
| НЕ РАВНО         | !=      | !=    |
| МЕНЬШЕ           | <       | <     |
| МЕНЬШЕ ИЛИ РАВНО | <=      | <=    |
| БОЛЬШЕ           | >       | >     |
| БОЛЬШЕ ИЛИ РАВНО | >=      | >=    |
| СОДЕРЖИТ         | >>      | &&    |

### Пример адресной строки, содержащей математический оператор

```http
http://url/.../query=ID,name?ID!=10?,,2,0
```

## Типы логических операторов

SQaLice поддерживает следующие логические операторы:

| Оператор  | SQaLice              | PG    |
| --------- | -------------------- | ----- |
| И         | *                    | AND   |
| ИЛИ       | Двойная прямая черта | OR    |

### Пример адресной строки запроса, содержащей оба оператора

```http
http://url/.../query=?(ID==8*title==Тест)||ID==10?
```

## Блок __fields__

Для получения всех полей из целевой SQL нужно передавать данный блок пустым.

### Пример адресной строки запроса с пустым блоком fields

```http
http://url/.../query=?ID==10||ID==8?ID,desc,2,0
```

Для получения конкретных полей, в поле fields необходимо передавать название поля в формате json с запятыми в качестве
разделителя и без пробелов.

```go
ID,title,createdAt?
```

### Пример адресной строки запроса для получения ID

```http
http://url/.../query=ID?ID>=1?ID,desc,10,0
```

При передаче некорректного названия поля, SQaLice вернет ошибку:

```go
"[SQaLice] Passed unexpected field name in select"
```

## Блок __conditions__

В данном блоке возможно указание условий получения записей. Допускается передача пустого блока __conditions__, в таком случае происходит выборка без дополнительных условий.

### Пример запроса без блока conditions

```http
http://url/.../query=ID,title,createdAt??ID,asc,10,0
```

Условия должны разделяться допустимыми [математическими операторами](#типы-математических-операторов) и не содержать пробелов.

### Пример использования простой условной конструкции

```go
?title==Тест?
```

Разделение нескольких условий осуществляется с помощью допустимых [логических операторов](#типы-логических-операторов), так же без пробелов. Кроме того могут быть использованы условные блоки в круглых скобках. Количество таких блоков не ограничено.

### Пример использования сложной условной конструкции

```go
(ID>8*ID<10)||title==Тест
```

#### ВАЖНО! При использовании сочетания условных блоков в скобках и без, блок без скобок должен указываться последним, как на приведенном выше примере. В противном случае запрос будет некорректным.

Для оператора *==* возможно указание нескольких значений через запятую, без пробелов. В таком случае происходит отбор значения по переданному массиву.

### Пример условной конструкции с выбором по массиву

```go
?ID==7,8,10?
```

При передаче условной конструкции с неверным логическим оператором, SQaLice вернет ошибку:

```go
"[SQaLice] Unsupported operator in condition"
```

При передаче условной конструкции с неверным названием поля, SQaLice вернет ошибку:

```go
"[SQaLice] Passed unexpected field name in condition"
```

## Блок __restrictions__

В данном блоке возможно указание ограничений конечной выборки. Допускается передача пустого блока ограничений, в таком случае SQaLice не накладывает дополнительных условий на выборку.

### Пример запроса без блока ограничений

```http
http://url/.../query=ID,title,createdAt?ID>1?
```

При заполнении ограничений в блоке __restrictions__ необходимо соблюдать порядок параметров блока, разделять их запятыми и не использовать пробелы

Порядок указания ограничений

| Параметр           | Допустимые значения          |
| ------------------ | ---------------------------- |
| Поле сортировки    | Поле, переданное в fieldsMap |
| Порядок сортировки | asc, desc                    |
| Лимит              | Целое число >= 0             |
| Оффсет             | Целое число >= 0             |

### Пример запроса со всеми параметрами ограничений

```http
http://url/.../query=ID,title,createdAt?ID>1?ID,asc,10,0
```

### Пример запроса с частью параметров (лимит, оффсет)

```http
http://url/.../query=ID,title,createdAt?ID>1?,,10,0
```

При передаче некорректного названия поля сортировки, SQaLice вернет ошибку:

```go
"[SQaLice] Unexpected selection order field"
```

При передаче некорректного порядка сортировки, SQaLice вернет ошибку:

```go
"[SQaLice] Unexpected selection order"
```

При передаче некорректного лимита, SQaLice вернет ошибку:

```go
"[SQaLice] Unexpected selection limit"
```

При передаче некорректного оффсета, SQaLice вернет ошибку:

```go
"[SQaLice] Unexpected selection offset"
```

## TO-DO

| TO-DO                                                             | Статус                       |
| ----------------------------------------------------------------- | ---------------------------- |
| Покрытие тестами                                                  | Выполнено                    |
| Обработка сложных условий (вложенность, дополнительные операторы) | Выполнено частично           |
| Снятие ограничений порядка условий                                | Не выполнено                 |

## Описание методов парсинга запроса

## Список полей

Функция *GetFieldsList* позволяет получать список полей, приведенных к формату запроса в базу.

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?ID==10||ID==8?ID,desc,2,0
```

Получаемый массив полей:

```go
["id", "title", "updated_at"]
```

При обработке строки запроса, не содержащей список полей, будет возвращен пустой массив:

```http
http://url/.../query=??
```

```go
[]
```

При запросе поля, не входящего в модель, компилятор вернет ошибку:

```go
"[SQaLice] Passed unexpected field name in select - field"
```

## Список условий

Функция *GetConditionsList* позволяет получить список условий, содержащихся в запросе. При этом элементы условий приводятся к формату запроса к БД.

Формат условной структуры:

| Имя       | Тип        | Описание                    |
| --------- | ---------- | --------------------------- |
| fieldName | string     | Название поля               |
| operator  | string     | Оператор                    |
| value     | interface  | Значение                    |
| isBracket | string     | Признак "Условие в скобках" |

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(title==testText)*ID!=8?ID,desc,2,0
```

Получаемый массив условий:

```json
[
    {
        "fieldName": "title",
        "operator": "=",
        "value": {"testText"},
        "isBracket": true
    },
    {
        "fieldName": "id",
        "operator": "!=",
        "value": {"8"},
        "isBracket": false
    },
]
```

При обработке строки запроса, не содержащей условий, будет возвращен пустой массив:

```http
http://url/.../query=??
```

```go
[]
```

Пример ошибки:

```go
"[SQaLice] Unsupported operator in condition - <<"
```

## Условие по названию

Функция *GetConditionByName* позволяет получить первое по порядку условие по с переданным названием поля, содержащегося в запросе. При этом элементы условия приводятся к формату запроса к БД.

Формат условной структуры:

| Имя       | Тип        | Описание                    |
| --------- | ---------- | --------------------------- |
| fieldName | string     | Название поля               |
| operator  | string     | Оператор                    |
| value     | interface  | Значение                    |
| isBracket | string     | Признак "Условие в скобках" |

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(ID==3)*ID!=8?ID,desc,2,0
```

Получаемый массив условий:

```json
{
    "fieldName": "id",
    "operator": "=",
    "value": {"3"},
    "isBracket": true
}
```

При обработке строки запроса, не содержащей условий, будет возвращен пустой массив:

```http
http://url/.../query=??
```

```go
[]
```

Пример ошибки:

```go
"[SQaLice] Unsupported operator in condition - <<"
```

## Поле сортировки

Функция *GetSortField* позволяет получить поле сортировки, содержащееся в запросе. При этом название поля приводится к формату запроса к БД.

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(ID==3)*ID!=8?ID,desc,2,0
```

Получаемое название поля:

```go
{
"id"
}
```

При обработке строки запроса, не содержащей ограничений, будет возвращена пустая строка:

```http
http://url/.../query=??
```

```go
""
```

Пример ошибки:

```go
"[SQaLice] Unsupported field in restrictions - field"
```

## Порядок сортировки

Функция *GetSortOrder* позволяет получить порядок сортировки, содержащийся в запросе.

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(ID==3)*ID!=8?ID,desc,2,0
```

Получаемый порядок сортировки:

```go
{"desc"}
```

При обработке строки запроса, не содержащей ограничений, будет возвращена пустая строка:

```http
http://url/.../query=??
```

```go
""
```

Пример ошибки:

```go
"[SQaLice] Unsupported field in restrictions - field"
```

## Лимит

Функция *GetLimit* позволяет получить лимит, содержащийся в запросе.

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(ID==3)*ID!=8?ID,desc,2,0
```

Получаемый порядок сортировки:

```go
2
```

При обработке строки запроса, не содержащей ограничений, будет возвращен nil:

```http
http://url/.../query=??
```

```go
nil
```

Пример ошибки:

```go
"[SQaLice] Invalid negative selection limit - -1"
```

## Отступ

Функция *GetOffset* позволяет получить отступ, содержащийся в запросе.

Пример строки запроса:

```http
http://url/.../query=ID,title,updatedAt?(ID==3)*ID!=8?ID,desc,2,0
```

Получаемый порядок сортировки:

```go
0
```

При обработке строки запроса, не содержащей ограничений, будет возвращен nil:

```http
http://url/.../query=??
```

```go
nil
```

Пример ошибки:

```go
"[SQaLice] Invalid negative selection offset - -1"
```

| TO-DO                                 | Статус        |
| ------------------------------------- | ------------- |
| Покрытие тестами                      | Выполнено     |
| Расширение получения условий запроса  | Не выполнено  |
