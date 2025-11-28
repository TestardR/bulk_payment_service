package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_ProcessBulkTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		bulkTransfer  BulkTransfer
		mockSetup     func(*MockAccountRepository)
		expectedError error
	}{
		{
			name: "successful bulk transfer",
			bulkTransfer: BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []Transfer{
					{
						CounterpartyName: "Bip Bip",
						CounterpartyIBAN: "EE383680981021245685",
						CounterpartyBIC:  "CRLYFRPPTOU",
						AmountCents:      1450,
						Currency:         "EUR",
						Description:      "Test transfer",
					},
					{
						CounterpartyName: "Bugs Bunny",
						CounterpartyIBAN: "FR0010009380540930414023042",
						CounterpartyBIC:  "RNJZNTMC",
						AmountCents:      99900,
						Currency:         "EUR",
						Description:      "Another transfer",
					},
				},
			},
			mockSetup: func(m *MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := NewMockAccountRepository(ctrl)

						account := Account{
							ID:           1,
							BalanceCents: 10000000,
						}

						expectedTransfers := []Transfer{
							{
								BankAccountID:    1, // bank_account_id set by service
								CounterpartyName: "Bip Bip",
								CounterpartyIBAN: "EE383680981021245685",
								CounterpartyBIC:  "CRLYFRPPTOU",
								AmountCents:      1450,
								Currency:         "EUR",
								Description:      "Test transfer",
							},
							{
								BankAccountID:    1, // bank_account_id set by service
								CounterpartyName: "Bugs Bunny",
								CounterpartyIBAN: "FR0010009380540930414023042",
								CounterpartyBIC:  "RNJZNTMC",
								AmountCents:      99900,
								Currency:         "EUR",
								Description:      "Another transfer",
							},
						}

						expectedAccount := Account{
							ID:           1,
							BalanceCents: 9898650, // 10000000 - 1450 - 99900
						}

						mockRepo.EXPECT().
							GetAccountByID(context.Background(), "FR10474608000002006107XXXXX", "OIVUSCLQXXX").
							Return(account, nil)

						mockRepo.EXPECT().
							UpdateBalance(context.Background(), expectedAccount).
							Return(nil)

						mockRepo.EXPECT().
							AddTransfers(context.Background(), expectedTransfers).
							Return(nil)

						return cb(mockRepo)
					}).
					Times(1)
			},
			expectedError: nil,
		},
		{
			name: "empty transfer list returns nil",
			bulkTransfer: BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers:        []Transfer{},
			},
			mockSetup:     func(m *MockAccountRepository) {},
			expectedError: nil,
		},
		{
			name: "insufficient funds returns error",
			bulkTransfer: BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []Transfer{
					{
						CounterpartyName: "Bip Bip",
						CounterpartyIBAN: "EE383680981021245685",
						CounterpartyBIC:  "CRLYFRPPTOU",
						AmountCents:      10000000,
						Currency:         "EUR",
						Description:      "Large transfer",
					},
				},
			},
			mockSetup: func(m *MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := NewMockAccountRepository(ctrl)

						account := Account{
							ID:           1,
							BalanceCents: 5000,
						}
						mockRepo.EXPECT().
							GetAccountByID(context.Background(), "FR10474608000002006107XXXXX", "OIVUSCLQXXX").
							Return(account, nil)

						return cb(mockRepo)
					}).
					Times(1)
			},
			expectedError: ErrInsufficientFunds,
		},
		{
			name: "account not found error propagates",
			bulkTransfer: BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []Transfer{
					{
						CounterpartyName: "Bip Bip",
						CounterpartyIBAN: "EE383680981021245685",
						CounterpartyBIC:  "CRLYFRPPTOU",
						AmountCents:      1450,
						Currency:         "EUR",
						Description:      "Test",
					},
				},
			},
			mockSetup: func(m *MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := NewMockAccountRepository(ctrl)

						accountNotFoundErr := errors.New("account not found")
						mockRepo.EXPECT().
							GetAccountByID(context.Background(), "FR10474608000002006107XXXXX", "OIVUSCLQXXX").
							Return(Account{}, accountNotFoundErr)

						return cb(mockRepo)
					}).
					Times(1)
			},
			expectedError: errors.New("account not found"),
		},
		{
			name: "update balance error propagates",
			bulkTransfer: BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []Transfer{
					{
						CounterpartyName: "Bip Bip",
						CounterpartyIBAN: "EE383680981021245685",
						CounterpartyBIC:  "CRLYFRPPTOU",
						AmountCents:      1450,
						Currency:         "EUR",
						Description:      "Test",
					},
				},
			},
			mockSetup: func(m *MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := NewMockAccountRepository(ctrl)

						account := Account{
							ID:           1,
							BalanceCents: 10000000,
						}

						expectedAccount := Account{
							ID:           1,
							BalanceCents: 9998550,
						}

						mockRepo.EXPECT().
							GetAccountByID(context.Background(), "FR10474608000002006107XXXXX", "OIVUSCLQXXX").
							Return(account, nil)

						dbError := errors.New("database connection error")
						mockRepo.EXPECT().
							UpdateBalance(context.Background(), expectedAccount).
							Return(dbError)

						return cb(mockRepo)
					}).
					Times(1)
			},
			expectedError: errors.New("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := NewMockAccountRepository(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			service := NewService(mockRepo)
			err := service.ProcessBulkTransfer(context.Background(), tt.bulkTransfer)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
