package gateway

import (
	analyticsv1 "github.com/DekuMidBak/gofintracker/gen/go/analytics/v1"
	transactionv1 "github.com/DekuMidBak/gofintracker/gen/go/transaction/v1"
	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
)

type Clients struct {
	Users        userv1.UserServiceClient
	Transactions transactionv1.TransactionServiceClient
	Analytics    analyticsv1.AnalyticsServiceClient
}
