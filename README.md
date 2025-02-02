# web3_draft

# Игра-викторина для web3
Основная концепция проекта заключается в создании игрового приложения на базе технологий Web3, где пользователи могут создавать комнаты для викторин и взаимодействовать с другими участниками. Хост комнаты задает вопрос, на который участники должны ответить. Каждый участник, подключившийся к комнате, оплачивает небольшую комиссию, а также платит за отправку своего ответа. После того как все ответы собраны, они становятся видимыми для всех, и участники могут голосовать за лучший ответ, выбирая тот, который, по их мнению, наиболее правильный или интересный. Лучшие ответы получают вознаграждения в токенах.

Игровой процесс организован в виде раундов: хост запускает каждый новый раунд с новым вопросом, и все участники могут участвовать в голосовании и давать свои ответы. Контракт отслеживает участие, проверяет готовность всех к следующему раунду и автоматически распределяет токены за лучшие ответы. В конце игры хост и участники получают вознаграждения, и все данные становятся неизменяемыми, **гарантируя честность и прозрачность** игры благодаря использованию блокчейн-технологий.
# Основная логика:

 1. Создание комнаты: хост создает комнату с указанием количества победителей раунда и игры в целом (например, 1 или 3), после чего ему возвращается id созданной комнаты. Для создания комнаты хосту необходимо оплатить комиссию, но во время игры ему будет доступна возможность получать токены за проведение игры. 
 2. Присоединение участников: хост передает участникам id созданной комнаты, после чего они заходят в нее, ждут подключения других игроков и начала игры. При подключении в комнату контракт списывает комиссию за свою работу.
 3. Начало раунда: хост создает вопрос. Для создания вопроса необходимо прикрепить данные для него. После создания вопроса, он отсылается всем привязанным к комнате юзерам, и начинается прием ответов. 
 4. Получение ответов: прием ответов заканчивается когда хост сам его завершает командой. С этого момента вопрос считается закрытым, и начинается открытие всех ответов пользователей. Если пользователь не успел ответить, то его ответ считается пустым и не участвует в голосовании. Если пользователь не ответил на вопрос или ему не хватило токенов на отправку ответа, то он считается выбывшим и больше не имеет права на посылку ответов и голосование (как бы становится наблюдателем, и с этого момента ему просто отсылается текущий статус игры, без возможности вмешиваться в ее процесс).
 5. Голосование: получив все ответы, у пользователя появляется возможность проголосовать за лучший по его мнению ответ. Голосовать можно до начала следующего раунда, иначе запрос будет отклонен.
 6. Завершение раунда: хост завершает раунд командой получения победителя раунда. Резульаты отправляются игрокам, а также распределяются вознаграждения за раунд между хостом, который берет больший процент, и непосредственно между выигравшими участниками в этом раунде, а средства на вознаграждения берутся из кошелька контракта комнаты, на который приходят начисления во время игры за создание вопросов и ответов на них. Если же хост хочет закончить игру, то он может сделать это только после этого шага и до начала следующего вопроса.
 7. Завершение игры: как только хост завершает игру, всем участникам отсылается топ победителей всей игры, распределяются награды. После успешного завершения игры хосту достается комиссия за проведение игры (остается на кошельке контракта money.go, откуда хост может вывести токены) а остальной процент отдается победителям в игре.

## Использование NFT токенов

Наш проект использует технологии Web3 и NFT, что предоставляет хостам возможность подчеркнуть **индивидуальность** своих игр. Каждая созданная хостом комната отличается уникальным стилем вопросов автора, который становится их “визитной карточкой”. Эти вопросы представлены в виде NFT, что гарантирует их неповторимость и защиту от копирования.

Хосты привлекают игроков в свои комнаты благодаря авторским вопросам, которые не встречаются в других играх и имеют больший спрос. Чем популярнее становится хост среди игроков, тем ценнее становятся его вопросы. В результате, у хостов появляется возможность создать собственную “экономику”:

- Покупка и продажа вопросов:
 хосты могут продавать свои уникальные вопросы другим создателям комнат, предоставляя возможность перенести успешные концепции в новые игры.
- Монетизация популярности:
 популярные хосты могут зарабатывать на продаже вопросов или использовании их в “коллаборациях” с другими комнатами, привлекая дополнительный трафик.

Эта механика способствует развитию игрового сообщества и стимулирует творчество среди хостов. Игроки, в свою очередь, получают разнообразный и уникальный опыт благодаря оригинальным стилям вопросов, создаваемых с помощью NFT.


## Описание игры:
![Описание основных действующих лиц и их действий](https://git.frostfs.info/nastyxxaavs/web3_draft/src/branch/master/schemes/Web3-Jackbox-uc.jpg)


![Процесс вызова метода игры (общий вид)](https://git.frostfs.info/nastyxxaavs/web3_draft/src/branch/master/schemes/Screenshot%202025-01-17%20174634.png)

## Список команд, используемых в игре:
#### host: 
- createRoom(host, countRoundWinners, countGameWinners) 
- startGame(roomId)
- askQuestion(roomId, tokenId), так как вопросы хоста представляются в виде уникальных NFT-токенов, то мы передаем их ID
- endQuestion(roomId)
- getRoundWinner(roomId)
- manuallyFinishGame(roomId)
#### player: 
- joinRoom(roomId)
- confirmReadiness(roomId)
- sendAnswer(roomId, text)
- voteAnswer(roomId, answerIdx), где answerIdx - индекс для сохранения порядка, в котором пишутся ответы при отправке игроками
- voteToFinishGame(roomId)