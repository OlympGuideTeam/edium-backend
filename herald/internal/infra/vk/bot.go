package vk

import (
	"fmt"
	"herald/internal/config"

	"github.com/SevereCloud/vksdk/v2/api"
	lp "github.com/SevereCloud/vksdk/v2/longpoll-bot"
)

func New(cfg config.VKConfig) (*api.VK, *lp.LongPoll, error) {
	vkAPI := api.NewVK(cfg.GroupToken)
	longPoll, err := lp.NewLongPoll(vkAPI, cfg.GroupID)
	if err != nil {
		return nil, nil, fmt.Errorf("create vk longpoll: %w", err)
	}
	return vkAPI, longPoll, nil
}
