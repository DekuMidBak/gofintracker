package grpc

import (
	"context"
	"errors"
	"time"

	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	"github.com/DekuMidBak/gofintracker/internal/transaction"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	CreateCategory(ctx context.Context, params transaction.CreateCategoryParams) (transaction.Category, error)
	ListCategories(ctx context.Context, filter transaction.ListCategoriesFilter) ([]transaction.Category, error)
	CreateTransaction(ctx context.Context, params transaction.CreateTransactionParams) (transaction.Transaction, error)
	ListTransactions(ctx context.Context, filter transaction.ListTransactionsFilter) (transaction.ListTransactionsResult, error)
	GetBalance(ctx context.Context, userID string) ([]transaction.CurrencyBalance, error)
}

type Server struct {
	transactionv1.UnimplementedTransactionServiceServer

	service Service
}

var _ transactionv1.TransactionServiceServer = (*Server)(nil)

func NewServer(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreateCategory(
	ctx context.Context,
	req *transactionv1.CreateCategoryRequest,
) (*transactionv1.CreateCategoryResponse, error) {
	category, err := s.service.CreateCategory(ctx, transaction.CreateCategoryParams{
		UserID: req.GetUserId(),
		Name:   req.GetName(),
		Type:   fromProtoType(req.GetType()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionv1.CreateCategoryResponse{
		Category: toProtoCategory(category),
	}, nil
}

func (s *Server) ListCategories(
	ctx context.Context,
	req *transactionv1.ListCategoriesRequest,
) (*transactionv1.ListCategoriesResponse, error) {
	var categoryType *transaction.Type
	if req.Type != nil {
		converted := fromProtoType(req.GetType())
		categoryType = &converted
	}

	categories, err := s.service.ListCategories(ctx, transaction.ListCategoriesFilter{
		UserID: req.GetUserId(),
		Type:   categoryType,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionv1.ListCategoriesResponse{
		Categories: toProtoCategories(categories),
	}, nil
}

func (s *Server) CreateTransaction(
	ctx context.Context,
	req *transactionv1.CreateTransactionRequest,
) (*transactionv1.CreateTransactionResponse, error) {
	created, err := s.service.CreateTransaction(ctx, transaction.CreateTransactionParams{
		UserID:      req.GetUserId(),
		CategoryID:  req.GetCategoryId(),
		Type:        fromProtoType(req.GetType()),
		Amount:      req.GetAmount(),
		Currency:    req.GetCurrency(),
		Description: req.GetDescription(),
		OccurredAt:  timestampToTime(req.GetOccurredAt()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionv1.CreateTransactionResponse{
		Transaction: toProtoTransaction(created),
	}, nil
}

func (s *Server) ListTransactions(
	ctx context.Context,
	req *transactionv1.ListTransactionsRequest,
) (*transactionv1.ListTransactionsResponse, error) {
	var categoryID *string
	if req.CategoryId != nil {
		categoryIDValue := req.GetCategoryId()
		categoryID = &categoryIDValue
	}

	var transactionType *transaction.Type
	if req.Type != nil {
		converted := fromProtoType(req.GetType())
		transactionType = &converted
	}

	from := optionalTimestampToTime(req.GetFrom())
	to := optionalTimestampToTime(req.GetTo())
	result, err := s.service.ListTransactions(ctx, transaction.ListTransactionsFilter{
		UserID:     req.GetUserId(),
		From:       from,
		To:         to,
		CategoryID: categoryID,
		Type:       transactionType,
		Limit:      int(req.GetLimit()),
		Offset:     int(req.GetOffset()),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionv1.ListTransactionsResponse{
		Transactions: toProtoTransactions(result.Transactions),
		TotalCount:   int32(result.TotalCount),
	}, nil
}

func (s *Server) GetBalance(
	ctx context.Context,
	req *transactionv1.GetBalanceRequest,
) (*transactionv1.GetBalanceResponse, error) {
	balances, err := s.service.GetBalance(ctx, req.GetUserId())
	if err != nil {
		return nil, mapError(err)
	}

	return &transactionv1.GetBalanceResponse{
		Balances: toProtoBalances(balances),
	}, nil
}

func fromProtoType(protoType transactionv1.TransactionType) transaction.Type {
	switch protoType {
	case transactionv1.TransactionType_TRANSACTION_TYPE_INCOME:
		return transaction.TypeIncome
	case transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE:
		return transaction.TypeExpense
	default:
		return transaction.Type("")
	}
}

func toProtoType(transactionType transaction.Type) transactionv1.TransactionType {
	switch transactionType {
	case transaction.TypeIncome:
		return transactionv1.TransactionType_TRANSACTION_TYPE_INCOME
	case transaction.TypeExpense:
		return transactionv1.TransactionType_TRANSACTION_TYPE_EXPENSE
	default:
		return transactionv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func toProtoCategory(category transaction.Category) *transactionv1.Category {
	return &transactionv1.Category{
		Id:        category.ID,
		UserId:    category.UserID,
		Name:      category.Name,
		Type:      toProtoType(category.Type),
		CreatedAt: timestamppb.New(category.CreatedAt),
	}
}

func toProtoCategories(categories []transaction.Category) []*transactionv1.Category {
	result := make([]*transactionv1.Category, 0, len(categories))
	for _, category := range categories {
		result = append(result, toProtoCategory(category))
	}

	return result
}

func toProtoTransaction(item transaction.Transaction) *transactionv1.Transaction {
	return &transactionv1.Transaction{
		Id:          item.ID,
		UserId:      item.UserID,
		CategoryId:  item.CategoryID,
		Type:        toProtoType(item.Type),
		Amount:      item.Amount,
		Currency:    item.Currency,
		Description: item.Description,
		OccurredAt:  timestamppb.New(item.OccurredAt),
		CreatedAt:   timestamppb.New(item.CreatedAt),
	}
}

func toProtoTransactions(items []transaction.Transaction) []*transactionv1.Transaction {
	result := make([]*transactionv1.Transaction, 0, len(items))
	for _, item := range items {
		result = append(result, toProtoTransaction(item))
	}

	return result
}

func toProtoBalances(balances []transaction.CurrencyBalance) []*transactionv1.CurrencyBalance {
	result := make([]*transactionv1.CurrencyBalance, 0, len(balances))
	for _, balance := range balances {
		result = append(result, &transactionv1.CurrencyBalance{
			Currency:      balance.Currency,
			IncomeAmount:  balance.IncomeAmount,
			ExpenseAmount: balance.ExpenseAmount,
			BalanceAmount: balance.BalanceAmount,
		})
	}

	return result
}

func timestampToTime(timestamp *timestamppb.Timestamp) time.Time {
	if timestamp == nil {
		return time.Time{}
	}

	return timestamp.AsTime()
}

func optionalTimestampToTime(timestamp *timestamppb.Timestamp) *time.Time {
	if timestamp == nil {
		return nil
	}

	value := timestamp.AsTime()
	return &value
}

func mapError(err error) error {
	switch {
	case errors.Is(err, transaction.ErrInvalidUserID),
		errors.Is(err, transaction.ErrInvalidCategoryID),
		errors.Is(err, transaction.ErrInvalidCategoryName),
		errors.Is(err, transaction.ErrInvalidType),
		errors.Is(err, transaction.ErrInvalidAmount),
		errors.Is(err, transaction.ErrInvalidCurrency),
		errors.Is(err, transaction.ErrInvalidOccurredAt),
		errors.Is(err, transaction.ErrInvalidTimeRange),
		errors.Is(err, transaction.ErrInvalidPagination),
		errors.Is(err, transaction.ErrInvalidID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, transaction.ErrDuplicateCategory):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, transaction.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
