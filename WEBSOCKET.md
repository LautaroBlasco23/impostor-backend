# WebSocket Integration

The game server includes real-time WebSocket support for live game updates.

## Connection

Connect to the WebSocket endpoint:
```
ws://localhost:3000/ws/{userId}?roomId={roomId}
```

Parameters:
- `userId`: The user's ID (from user creation)
- `roomId`: (Optional) The room ID to join immediately

## Event Types

All events follow this structure:
```json
{
  "type": "event_type",
  "room_id": "room-uuid",
  "payload": {}
}
```

### Server Events (Received by Client)

#### `user_joined`
Broadcast when a user joins the room
```json
{
  "type": "user_joined",
  "room_id": "room-123",
  "payload": {
    "user_id": "user-456",
    "nickname": "Player1",
    "room_id": "room-123"
  }
}
```

#### `user_left`
Broadcast when a user leaves the room
```json
{
  "type": "user_left",
  "room_id": "room-123",
  "payload": {
    "user_id": "user-456",
    "nickname": "Player1"
  }
}
```

#### `user_ready`
Broadcast when a user toggles ready status
```json
{
  "type": "user_ready",
  "room_id": "room-123",
  "payload": {
    "user_id": "user-456",
    "nickname": "Player1",
    "is_ready": true
  }
}
```

#### `category_set`
Broadcast when room leader sets the category
```json
{
  "type": "category_set",
  "room_id": "room-123",
  "payload": {
    "category": "animals",
    "room_id": "room-123"
  }
}
```

#### `game_started`
Broadcast when the game starts
```json
{
  "type": "game_started",
  "room_id": "room-123",
  "payload": {
    "game_id": "game-789",
    "category": "animals",
    "round_number": 1
  }
}
```

#### `user_voted`
Broadcast when a user submits a vote
```json
{
  "type": "user_voted",
  "room_id": "room-123",
  "payload": {
    "voter_id": "user-456",
    "voter_name": "Player1",
    "target_id": "user-789",
    "target_name": "Player2",
    "votes_cast": 2,
    "total_players": 5
  }
}
```

#### `user_eliminated`
Broadcast when a user is eliminated
```json
{
  "type": "user_eliminated",
  "room_id": "room-123",
  "payload": {
    "user_id": "user-789",
    "nickname": "Player2",
    "was_impostor": false,
    "vote_count": 3
  }
}
```

#### `game_won`
Broadcast when players win (impostor eliminated)
```json
{
  "type": "game_won",
  "room_id": "room-123",
  "payload": {
    "impostor_id": "user-999",
    "impostor_name": "Impostor",
    "message": "Players win! Impostor was the impostor!"
  }
}
```

#### `game_lost`
Broadcast when impostor wins
```json
{
  "type": "game_lost",
  "room_id": "room-123",
  "payload": {
    "impostor_id": "user-999",
    "impostor_name": "Impostor",
    "message": "Impostor wins! Not enough players remaining!"
  }
}
```

#### `room_update`
Broadcast for general room updates (new round, etc.)
```json
{
  "type": "room_update",
  "room_id": "room-123",
  "payload": {
    "round_number": 2,
    "message": "Player2 was not the impostor. Continue playing!"
  }
}
```

## Client Example (JavaScript)

```javascript
const userId = "user-123";
const roomId = "room-456";
const ws = new WebSocket(`ws://localhost:3000/ws/${userId}?roomId=${roomId}`);

ws.onopen = () => {
  console.log("Connected to game server");
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'user_joined':
      console.log(`${data.payload.nickname} joined the room`);
      break;
      
    case 'user_ready':
      console.log(`${data.payload.nickname} is ${data.payload.is_ready ? 'ready' : 'not ready'}`);
      break;
      
    case 'user_voted':
      console.log(`${data.payload.voter_name} voted for ${data.payload.target_name}`);
      console.log(`Votes: ${data.payload.votes_cast}/${data.payload.total_players}`);
      break;
      
    case 'user_eliminated':
      if (data.payload.was_impostor) {
        console.log(`${data.payload.nickname} was the impostor! Players win!`);
      } else {
        console.log(`${data.payload.nickname} was eliminated (not impostor)`);
      }
      break;
      
    case 'game_won':
      console.log(data.payload.message);
      break;
      
    case 'game_lost':
      console.log(data.payload.message);
      break;
  }
};

ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};

ws.onclose = () => {
  console.log("Disconnected from game server");
};
```

## Game Flow with WebSockets

1. **User creates account** → No WS event
2. **User joins room** → `user_joined` broadcast to room
3. **User toggles ready** → `user_ready` broadcast to room
4. **Leader sets category** → `category_set` broadcast to room
5. **Game starts** → `game_started` broadcast to room
6. **User votes** → `user_voted` broadcast to room
7. **All votes cast** → `user_eliminated` broadcast, then either:
   - `game_won` if impostor eliminated
   - `game_lost` if impostor survives with ≤1 other player
   - `room_update` if game continues to next round

## Implementation Notes

- WebSocket connections are managed per-room
- Only users in the same room receive broadcasts
- Connection is maintained throughout the game session
- Automatic cleanup on disconnect
- All events are JSON-encoded
