package pickup

import "sync"

// Queue is a queue of players.
type Queue struct {
	RequiredPlayers int
	Players         []*Player
	mutex           sync.Mutex
}

// Enqueue adds a player to the queue. If the queue becomes filled when the player is added, a room is created and returned.
func (queue *Queue) Enqueue(player *Player) *Room {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	queue.Players = append(queue.Players, player)

	if len(queue.Players) == queue.RequiredPlayers {
		room := new(Room)
		room.Size = queue.RequiredPlayers
		for i := 0; i < queue.RequiredPlayers; i++ {
			room.AddPlayer(queue.Dequeue())
		}
		return room
	}

	return nil
}

// EnqueueTeam adds a group of players to the queue. If the queue becomes filled when the team is added, a room is created and returned.
func (queue *Queue) EnqueueTeam(players []*Player) *Room {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	if len(queue.Players)+len(players) >= queue.RequiredPlayers {
		room := new(Room)
		room.Size = queue.RequiredPlayers
		for i := 0; i < len(players); i++ {
			room.AddPlayer(players[i])
		}
		for i := len(players); i < queue.RequiredPlayers; i++ {
			room.AddPlayer(queue.Dequeue())
		}
		return room
	}

	queue.Players = append(queue.Players, players...)

	return nil
}

// Dequeue removes a player from the front of the queue.
func (queue *Queue) Dequeue() *Player {
	if len(queue.Players) == 0 {
		return nil
	}

	player := queue.Players[0]
	copy(queue.Players[0:], queue.Players[1:])
	queue.Players[len(queue.Players)-1] = nil
	queue.Players = queue.Players[:len(queue.Players)-1]
	return player
}

// Remove removes a player from anywhere within the queue.
func (queue *Queue) Remove(player *Player) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()

	for i, p := range queue.Players {
		if p == player {
			if i == 0 {
				queue.Dequeue()
			} else if i == len(queue.Players)-1 {
				copy(queue.Players[i:], queue.Players[i+1:])
				queue.Players[len(queue.Players)-1] = nil
				queue.Players = queue.Players[:len(queue.Players)-1]
			} else {
				queue.Players[len(queue.Players)-1] = nil
				queue.Players = queue.Players[:len(queue.Players)-1]
			}
		}
	}
}

// Top gets the player at the front of the queue.
func (queue *Queue) Top() *Player {
	if len(queue.Players) == 0 {
		return nil
	}
	return queue.Players[0]
}

// Len returns the length of the queue.
func (queue *Queue) Len() int {
	return len(queue.Players)
}
