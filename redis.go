package main

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// server_list: [s1:<room_count>, s2:<room_count>, s3]
// <room_id> : {<server name>, <num_players>}
// <server_name> : [room_id, roomid2]

func initRedis() *redis.Client {

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return rdb
}

// Find an underutilized server and return url
// Used when trying to create a room but node client is connecting to
// is full
func (a App) findServer() (string, error) {

	ctx := context.Background()
	res, err := a.Redis.ZRange(ctx, "server_list", 0, 1).Result()
	if err != nil {
		return "", err
	}
	return res[0], err
}

// Update number of rooms in a server
// ZINCRBY server_list <server_id> 1
// Used after room created succesfully
func (a App) updateNumRooms(roomId string, incrBy float64) error {
	ctx := context.Background()
	res := a.Redis.ZIncrBy(ctx, "server_list", incrBy, roomId)
	return res.Err()
}

// Find the server a room is in. Return server url
func (a App) findRoom(roomId string) (string, error) {

	ctx := context.Background()
	res, err := a.Redis.HGet(ctx, roomId, "server_name").Result()
	if err != nil {
		return "", err
	}
	return res, nil
}

// Add server name to list of available servers
// Done at node initialization
func (a App) addSelfToServerList(name string) error {

	ctx := context.Background()
	_, err := a.Redis.ZAdd(ctx, "server_list", redis.Z{Score: 0, Member: name}).Result()

	return err
}

// Add room hashset to redis
// Done on room creation or room join
func (a App) setRoomInfo(roomId string, numPlayers int) error {

	ctx := context.Background()
	_, err := a.Redis.HSet(ctx, roomId, "name", a.NodeName, "num_players", numPlayers).Result()
	return err
}

func (a App) deleteKey(key string) error {
	ctx := context.Background()
	_, err := a.Redis.Del(ctx, key).Result()
	return err
}
