## Описание функций, используемых в игре:
wallet1 - создает комнату

игроки:
wallet2, 
wallet3, 
wallet4, 
и далее

### Команды для деплоя контрактов комнаты и токенов:

```neo-go contract compile -i contract.go```

```neo-go contract deploy -i contract.nef -m contract.manifest.json -r http://localhost:30333 -w wallet1.json```

  
### Игровые команды

- *Создание комнаты хостом*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash createRoom [ countRoundWinners countGameWinners ]```

##### Аргументы метода: 

1. countRoundWinners - кол-во победителей раунда
2. countGameWinners -  кол-во победителей игры

- *Вход участников в комнату* (на примере игрока wallet2)

```neo-go contract invokefunction -r http://localhost:30333 -w wallet2.json -g gas_payment contractHash joinRoom [ roomId ]```

##### Аргументы метода: 

1. roomId - ID комнаты, которое заранее передано игроку вне данной системы

##### Аналогично для:

wallet3 joinRoom

wallet4 joinRoom

- *Подтверждение готовности к игре участниками*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet2.json -g gas_payment contractHash confirmReadiness [ roomId ]```

##### Аргументы метода: 
1. roomId - ID комнаты, которое заранее передано игроку вне данной системы

##### Аналогично для:

wallet3 confirmReadiness

wallet4 confirmReadiness

- *Запуск игры хостом*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash startGame [ roomId ]```

##### Аргументы метода: 

1. roomId - ID созданной комнаты

- *Публикация вопроса текущего раунда*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash askQuestion [ roomId tokenId ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты
2. tokenId - ID токена. Так как вопросы хоста представляются в виде уникальных NFT-токенов, то мы передаем их ID

- *Отправка ответа на вопрос*

```$ ./bin/neo-go contract invokefunction -r http://localhost:30333 -w wallet2.json -g gas_payment contractHash sendAnswer [ roomid text ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты
2. text - ответ на вопрос

##### Аналогично для:

wallet3 sendAnswer

wallet4 sendAnswer

- *Завершение принятие ответов (раунда)*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash endQuestion [ roomid ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты

- *Отдача голоса за лучший ответ*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet2.json -g gas_payment contractHash voteAnswer [ roomId answerIdx ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты
2. answerIdx - индекс ответа

##### Аналогично для:

wallet3 voteAnswer

wallet4 voteAnswer

- *Завершение раунда*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash getRoundWinner [ roomId ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты


*Далее повторяется игровой цикл:*

askQuestion..

sendAnswer..

endQuestion..

voteAnswer..

getRoundWinner..


- *Отдача голоса за завершение игры (без учета мнения хоста)*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet2.json -g gas_payment contractHash voteToFinishGame [ roomId ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты

##### Аналогично для:

wallet3 voteToFinishGame

wallet4 voteToFinishGame

- *Завершение игры*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash manuallyFinishGame [ roomId ]```

##### Аргументы метода: 
1. roomId - ID созданной комнаты

### Команды для взаимодействия с money.go, доступно только хосту

- *Получение баланса с игрового кошелька*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash GetBalance```


- *Вывод газа с игрового кошелька*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash HostWithdrawal [ amount ]```

### Команды для взаимодействия с nft.go

- *Получение списка своих NFT*

```neo-go contract invokefunction -r http://localhost:30333 -w wallet1.json -g gas_payment contractHash TokensOfList [ wallet ]```

- *покупка NFT с вопросом, поле sourceLink необязательное. Подразумевается, что оно для дополнительных данных к вопросу, ссылке* 

```neo-go wallet nep17 transfer -r http://localhost:30333 -w /path/to/wallet.json --from <sender_address> --to <money_adress> --amount 10 --token GAS '{"question":"What is Neo?", "sourceLink":"link<optional>"}' --await```

### Команды для взаимодействия с neo-go

- Проверка баланса

```neo-go wallet nep17 balance --token GAS -r http://localhost:30333 -w /path/to/frostfs-aio/wallets/wallet1.json```

- Компиляция и деплой

```neo-go contract compile -i nep11/contract.go -o nep11/contract.nef -m nep11/contract.manifest.json -c nep11/contract.yml```

```neo-go contract deploy -i nep11/contract.nef -m nep11/contract.manifest.json -r http://localhost:30333 -w /path/to/frostfs-aio/wallets/wallet1.json [ walletHash ]```

- Трансфер любого токена

```neo-go wallet nep17 transfer  -r http://localhost:30333 -w /path/to/frostfs-aio/wallets/wallet1.json --from walletHash --to walletHash --amount 20 --token GAS nft-name --await```

- Проверка любой транзакции в логах

```curl -s --data '{ "jsonrpc": "2.0", "id": 1, "method": "getapplicationlog", "params": ["<хэш транзакции>"] }' https://localhost:30333 | jq```
- neo-go util convert - для конвертации типов

![Обобщенное описание процесса игры](https://git.frostfs.info/nastyxxaavs/web3_draft/src/branch/master/schemes/web3_activity_diagram.jpg)





