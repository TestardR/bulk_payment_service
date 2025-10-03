package core_test

import (
	"context"
	"errors"
	"testing"

	"qonto/internal/core"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_ProcessBulkTransfer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		bulkTransfer  core.BulkTransfer
		mockSetup     func(*core.MockAccountRepository)
		expectedError error
	}{
		{
			name: "successful bulk transfer",
			bulkTransfer: core.BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []core.Transfer{
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
			mockSetup: func(m *core.MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(core.AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := core.NewMockAccountRepository(ctrl)

						account := core.Account{
							ID:           1,
							BalanceCents: 10000000,
						}

						expectedTransfers := []core.Transfer{
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

						expectedAccount := core.Account{
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
			bulkTransfer: core.BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers:        []core.Transfer{},
			},
			mockSetup:     func(m *core.MockAccountRepository) {},
			expectedError: nil,
		},
		{
			name: "insufficient funds returns error",
			bulkTransfer: core.BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []core.Transfer{
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
			mockSetup: func(m *core.MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(core.AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := core.NewMockAccountRepository(ctrl)

						account := core.Account{
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
			expectedError: core.ErrInsufficientFunds,
		},
		{
			name: "account not found error propagates",
			bulkTransfer: core.BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []core.Transfer{
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
			mockSetup: func(m *core.MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(core.AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := core.NewMockAccountRepository(ctrl)

						accountNotFoundErr := errors.New("account not found")
						mockRepo.EXPECT().
							GetAccountByID(context.Background(), "FR10474608000002006107XXXXX", "OIVUSCLQXXX").
							Return(core.Account{}, accountNotFoundErr)

						return cb(mockRepo)
					}).
					Times(1)
			},
			expectedError: errors.New("account not found"),
		},
		{
			name: "update balance error propagates",
			bulkTransfer: core.BulkTransfer{
				OrganizationBIC:  "OIVUSCLQXXX",
				OrganizationIBAN: "FR10474608000002006107XXXXX",
				Transfers: []core.Transfer{
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
			mockSetup: func(m *core.MockAccountRepository) {
				m.EXPECT().
					Atomic(context.Background(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, cb func(core.AccountRepository) error) error {
						ctrl := gomock.NewController(t)
						mockRepo := core.NewMockAccountRepository(ctrl)

						account := core.Account{
							ID:           1,
							BalanceCents: 10000000,
						}

						expectedAccount := core.Account{
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

			mockRepo := core.NewMockAccountRepository(ctrl)
			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo)
			}

			service := core.NewService(mockRepo)
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
