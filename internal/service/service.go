package service

import (
	"github.com/olgapark/avito-labs/internal/repository"
	"github.com/olgapark/avito-labs/internal/txmanager"
)

type Service struct {
	repo      *repository.Repository
	txManager txmanager.TxManager
}

func New(repo *repository.Repository, txManager txmanager.TxManager) *Service {
	return &Service{repo: repo, txManager: txManager}
}
