In progress ....

### Структура проекта
```
+---cmd
|   \---server
|           main.go                             # точка запуска
\---internal
    +---adapters                                 # уровень http:    HTTP -- service -- domain
    |   +---handler                              # http handlers (get - присоединение к чату и стриминг, post - отправка сообщения )
    |   |   |   get.go
    |   |   |   post.go
    |   |   |   post_parse_body.go               # парсинг body от POST {pubkey} {signature} {message}\n
    |   |   \---test                             # тесты уровня handlers. реализованы пока только для post
    |   |           post_mosk_test.go
    |   |           post_parse_body_test.go
    |   |           post_test.go
    |   |           test_helpers_test.go
    |   +---middleware                           # используемые middleware
    |   |       global_limiter.go                # защитник на лимиты. Лимитирует максимум запросов на весь сервер в секунду
    |   |       logger.go                        # изменения в логгере
    |   |       post_body_limit.go               # защитник на длину поля в post    
    |   \---stream
    |           chunked_streamer.go              # реализация http chunked streaming          
    +---app
    |       app.go                               # Сервер-приложение. Запускаем тут все в работу 
    +---config
    |       config.go                            #  Конфиг уровня app
    +---domain                                   # Доменная мрдель чата:  http -- service - DOMAIN
    |   |   bootstrap.go
    |   |   config.go
    |   |   const.go
    |   |   errors.go
    |   |   event.go
    |   |   event_ring_buf.go
    |   |   event_test.go
    |   |   export_test.go
    |   |   formatter.go
    |   |   member.go
    |   |   member_factory.go
    |   |   member_overflow_buf.go
    |   |   member_test.go
    |   |   message.go
    |   |   room.go
    |   |   room_factory.go
    |   |   room_history_store.go
    |   |   room_test.go
    |   |   validators.go
    |   \---crypto                              # криптография
    |           ed25519.go
    |           provider.go
    \---service                               # слой сервиса (оркестрация доменных сущностей):  http -- SERVICE - domain
            chat_service.go
            chat_service_join.go                 # Реализация присоединения к чату
            chat_service_lifecycle.go            # Реализация ЖЦ сервиса
            chat_service_send.go                 # Реализация отправки сообщения 
            chat_service_test.go
            config.go                            # Конфиг с параметрами влиящими на производительность комнат и лимиты
            connection_tracker.go                # Защитник на кол-во одновременных сессий (комнат)
            errors.go
            rate_limiter.go                      # Защитник RateLimit реализация (ограничитель на действия с одного IP действий с комнатами и сообщениями за период)
            room_rate_guard.go                   
            room_storage.go                      # InMemory хранилище комнат
            session_guard.go                     # Защитник, отправлять сообщения только с того IP откуда был Get
            stream_writer.go                     # Абстракция стрмингового писателя для чата
            types.go

```
