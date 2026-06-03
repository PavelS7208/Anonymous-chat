# Тестовый проект: Анонимный чат. Реализация Этап 3

ReadMe [Этапа 2](README_step_2_main.md)


## Оглавление

- [Архитектура](#архитектура)
- [Слои приложения](#слои-приложения)
- [Структура проекта](#структура-проекта)
- [Ключевые особенности реализации](#ключевые-особенности-реализации)
  - [1. Отделение жизненного цикла чата-сервера от бизнес-задач чата](#1-отделение-жизненного-цикла-чата-сервера-от-бизнес-задач-чата)
  - [2. Не блокирующая асинхронная отправка событий](#2-не-блокирующая-асинхронная-отправка-событий)
  - [3. Гарантия порядка отображения событий в чате](#3-гарантия-порядка-отображения-событий-в-чате)
  - [4. Вынесение Write-а стриминга за интерфейс](#4-вынесение-write-а-стриминга-за-интерфейс)
  - [5. Абстракция RoomStorage](#5-абстракция-roomstorage)
- [Бэклог](#бэклог)


### Архитектура
Проект реализован в стандартной трёхслойной архитектуре:

Handler -> Service -> Domain (Model) 

### Слои приложения

- **Handler** (HTTP-слой):
    - Регистрация маршрутов с использованием роутера chi
    - Парсинг HTTP-запросов 
    - Обработка HTTP-ошибок и статус-кодов
    - Структурная валидация входных данных
    - Защитники от атак реализованные на уровне middleware
    

- **Service** (слой оркестрации):
    - **Только инфраструктурные сущности** — вся базовая бизнес-логика в доменных сущностях
    - Координация между доменными сущностями
    - In-memory хранилище комнат-чатов
    - Entity Records для изоляции сущностей сервиса от доменных моделей
    - **НЕ содержит** бизнес-правил — они в доменных моделях
    - Работа защитников от атак работающих до вызова Бизнес-методов (стриминг в чате и broadcast сообщения)


- **Domain Model** (доменный слой):
    - **Room** — чат-комната со списком участников, работает с историей чата через интерфейс **event_history** 
    - **Member** — участник
    - **Event, Message, Bootstrap** — элементы, которые живут внутри чата
    - **Валидации** - встроены в методы или helpers
    -  Атомарность операций (mutex для конкурентного доступа) 
    - **Юнит-тесты без моков** — доменная логика тестируется напрямую


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
    +---domain                                   # Доменная мрдель чата:  http -- service - DOMAIN.  Тесты в процессе
    |   |   bootstrap.go                               # инициализирующее событие с приватним ключом (seed)
    |   |   config.go                                  # конфиг уровня домена
    |   |   const.go                                   #  константы из ТЗ
    |   |   errors.go
    |   |   event.go
    |   |   event_ring_buf.go                           # реализация кольцевого буфера, для истории и управления медленными клиентами
    |   |   event_test.go
    |   |   export_test.go
    |   |   formatter.go
    |   |   member.go                                   # участник чата и его фабрика
    |   |   member_factory.go
    |   |   member_overflow_buf.go                      # механизм дополнительного буфера для медленного клиента
    |   |   member_test.go
    |   |   message.go
    |   |   room.go                                     # комната и ее фабрика
    |   |   room_factory.go
    |   |   room_history_store.go                       # история, скрытая за интерфейс (текущая реализацяия или кольцевой буфер или slice )
    |   |   room_test.go
    |   |   validators.go                               # бизнесовая валидация
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

```


### Ключевые особенности реализации

#### 1. Отделение жизненного цикла чата-сервера от бизнес-задач чата

``` GO
type Starter interface {
  Start(ctx context.Context) error
}

type Closer interface {
  Close() error
}

type lifecycleEntry struct {
  starter Starter // nil если не умеет Start
  closer  Closer  // nil если не умеет Close
}

type ChatServiceLifecycle struct {
  entries []lifecycleEntry // порядок вызовов сохраняется
}


func NewChatService(cfg RoomServiceConfig, cp crypto.Provider, logger *slog.Logger) (*ChatService, *ChatServiceLifecycle) {
	cfg = cfg.withDefaults()

    ....
	rf := domain.NewRoomFactory(cfg.mng.room, domain.NewMemberFactory() )
	repo := newInMemoryRoomStorage(cfg.mng, rf)
	
	rGuard := newRoomRateGuard(cfg.limiter)
	connTr := newConnectionTracker(cfg.MaxConnectionsPerIP)
	sGuard := newSessionGuard(cfg.SessionAbsoluteTTL, cfg.SessionIdleTTL)

	svc := &ChatService{
        ....
	}

    // Регистрация сущностей для которых необходим запуск и очистка ресурсов
	lc := &ChatServiceLifecycle{}
	lc.Register(repo)
	lc.Register(rGuard)
	lc.Register(sGuard)

	return svc, lc
}

    // Использование в app(main)
    // Создаём сервис + контроллер жизненного цикла
    chatSvc, lifecycle := service.NewChatService(cfg.RoomService, cryptoProvider, logger)

    // chatSvc - ушел в http "ручки"
    getHandler := handler.NewGetHandler(chatSvc, logger)
    postHandler := handler.NewPostHandler(chatSvc, logger)


    // А ЖЦ управляется отдельно. Инициацализация необходимая, запуск сервисных горутин и т.п.
    bgCtx, bgCancel := context.WithCancel(ctx)
    lifecycle.Start(bgCtx)
	
    // Освобождение ресурсов для кого это нужно
    lifecycle.Close() // закрываем комнаты, участников, защитников .....
```

#### 2. Не блокирующая асинхронная отправка событий
Отправка событий всем участникам комнаты не блокирующая происходит в отдельной горутине. Введен буфер у комнаты для отправки событий участникам.  

Для каждого клиента-учавстника выделен свой собственный буфер (канал буферизированный), который сглаживает возможные проблемы на стороне клиента-участника при отправке ему сообщений. Порядок сообщений сохраняется.\

Если же буфер клиента заполнился, то такому клиенту дается "последний шанс", резервируется еще один буфер на основе ring buf (буфер с нулевыми аллокациями) и в случае заполнения и его только тогда клиент отключается как зависший.
  
```go
// Run - фоновая горутина fan-out рассылки.
// Читает из broadcast и асинхронно доставляет события участникам.
// Использует алгоритм "последний шанс" для клиента у которого буфер заполнился (доп массив overflow)
func (r *Room) Run() {
	for event := range r.broadcast {
		var deadIDs []MemberID  // Для сбора зависших, чтобы удалить их без лока

		r.mu.RLock()
		for _, m := range r.members {
			if event.IsSystem() && m.id == event.SenderID {
				continue
			}
			// Дренаж overflow в строгом FIFO-порядке (peek-then-pop)
			// Гарантирует: старое событие уходит первым, новые не могут его обогнать
			for {
				// Заглядываем в голову буфера, но НЕ удаляем событие
				ovEvent, ok := m.overflow.peek()
				if !ok {
					break // Буфер пуст
				}
				// Пытаемся доставить событие в основной канал
				select {
				case m.events <- ovEvent:
					// Успех: удаляем из буфера
					_, _ = m.overflow.pop()
					// Продолжаем цикл: пробуем доставить следующее событие в очереди
				default:
					// Неудача: основной канал всё ещё полон
					// Оставляем событие в голове буфера
					// Прерываем дренаж: если старое не прошло, новые тоже упрутся
					break
				}
			}
			// Обработка нового события (event) из широковещательной рассылки
			if m.overflow.len() > 0 {
				// Overflow не пуст
				// новое событие добавляем в хвост очереди
				// Оно будет доставлено после того, как уйдут все старые (гарантия FIFO)
				if !m.overflow.push(event) {
					// Буфер переполнен, клиент не справляется и с доп буфером - отключаем
					m.Close()
					deadIDs = append(deadIDs, m.id)
				}
			} else {
				// пробуем отправить новое событие из основного буфера (overflow - пуст)
				select {
				case m.events <- event:
					// Доставлено успешно
				default:
					// Основной канал полон - кладём новое событие в overflow
					if !m.overflow.push(event) {
						// Переполнение overflow - клиент не справляется, отключаем
						m.Close()
						deadIDs = append(deadIDs, m.id)
					}
				}
			}
		}
		r.mu.RUnlock()
		// удаляем мертвых
	}
}

```
#### 3. Гарантия порядка отображения событий в чате

Так как используется Unix время в секундах, то для нескольких событий может быть одно время, т.е. необходим дополнительный механизм гарантирующий сохранение порядка событий при выводе в чат-комнате клиенту.\
Введен seq - монотонно увеличивающийся счетчик в комнате и проверки на < > при выводе

После получения окна из 10 последних событий из истории и отправки их новому присоединившемуся клиенту, происходит доотправка событий, которые могли прийти за время отправки полученных данных (snapshot).

#### 4. Вынесение Write-а стриминга за интерфейс
Заложена масштабируемость для простой замены chunked на иной другой формат отправки сообщений в чате.

Форматирование согласно требованиям вынесено за отдельно в Marshal для замены в одном месте, в случае изменения требований.


```go
// WireMessage — контракт для доменных типов, поддерживающих сериализацию в проводной протокол.
// Экспортируется для использования в адаптерах
type WireMessage interface {
    Marshal() []byte
}

// ChatWriter — контракт для записи данных в транспорт (авто-flush где-то внутри реализации)
type ChatWriter interface {
    Write(ctx context.Context, msg WireMessage) error
}

func (s *ChatService) Stream(ctx context.Context, session *JoinSession, w ChatWriter) error {

	....
		
	// Отправка bootstrap с приватным ключом — первый Write, HTTP 200 фиксируется здесь
	if err := w.Write(hCtx, domain.NewBootstrap(session.member.ID(), session.privateSeed)); err != nil {
		return err
	}
	session.privateSeed = nil // приватный ключ больше не нужен — обнуляем

	// Отправка снапшота истории комнаты 
	if err := sendSnapshot(hCtx, session.snapshot, w); err != nil {
            return err
	}

	// Handshake завершён — активируем участника и оповещаем комнату
	session.member.SetActivated()
	session.room.Broadcast(domain.NewJoinEvent(session.member.ID()))

	// Основной цикл стриминга событий
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-session.member.Events():
			if !ok {
				return nil
			}
			// Фильтр дублей: пропускаем события уже вошедшие в снапшот
			if event.Seq <= session.lastSeq {
				continue
			}
			if err := w.Write(ctx, event); err != nil {
				s.logger.Error("client write error", "err", err, "room", session.room.Name())
				return err
			}
			s.logger.Debug("client write event", "room", session.room.Name())

		}
	}
}

// sendSnapshot отправляет срез событий через StreamWriter
func sendSnapshot(ctx context.Context, events []domain.Event, w ChatWriter) error {
    for _, evt := range events {
        if err := w.Write(ctx, evt); err != nil {
            return err
        }
    }
    return nil
}


```

#### 5. Абстракция RoomStorage

Хранение и доступ с комнатам-чатам вынесено за интерфейс. Готово для перехода с InMemory на иные формы реализации хранилища/репозитария.

```go
type roomStorage interface {
	Get(ctx context.Context, roomName string) (*domain.Room, error)
	GetOrCreate(ctx context.Context, roomName string) (*domain.Room, error)
	Count() int
}

```




### Бэклог
