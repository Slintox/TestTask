# Тестовое задание

Реализованы чтение и запись по одному символу между двумя файлами. 
Такой подход крайне медленный, поэтому реализованы также буферизованные версии функций записи и чтения (используются для отладки).
Передача от читающей горутины к пишущей происходит по одному байту.
Для чтения\записи по одному символу с помощью буферизованных функций нужно выставить значения 
readBlockSize \ writeBlockSize как единицу.

В случае ошибки во время выполнения записи в файл, файл "теряется" (в файле сохраняются уже записанные данные, откат к предыдущей
версии не реализован).

Флаг -neg добавляет в поиск названия с отрицательными числами.

Доступные флаги:
```
-config-path [string]
    Path to the config file (default "configs/config.yml")
-neg
     Allow reading negative names
-rbs [int]
     The number of bytes read at a time (default 1)
-wbs [int]
     The number of bytes written at a time (default 1)
```