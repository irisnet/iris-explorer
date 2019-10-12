package service

import (
	"github.com/irisnet/explorer/backend/conf"
	"github.com/irisnet/explorer/backend/lcd"
	"github.com/irisnet/explorer/backend/logger"
	"github.com/irisnet/explorer/backend/orm/document"
	"github.com/irisnet/explorer/backend/types"
	"github.com/irisnet/explorer/backend/utils"
	"github.com/irisnet/explorer/backend/vo"
	"strconv"
	"math/big"
	"github.com/irisnet/irishub-sync/util/constant"
)

type AccountService struct {
	BaseService
}

func (service *AccountService) GetModule() Module {
	return Account
}

func (service *AccountService) Query(address string) (result vo.AccountVo) {
	prefix, _, _ := utils.DecodeAndConvert(address)
	if prefix == conf.Get().Hub.Prefix.ValAddr {
		self, delegated := delegatorService.QueryDelegation(address)
		result.Amount = utils.Coins{self}
		result.Deposits = delegated

	} else {
		res, err := lcd.Account(address)
		if err == nil {
			var amount utils.Coins
			for _, coinStr := range res.Coins {
				coin := utils.ParseCoin(coinStr)
				amount = append(amount, coin)
			}
			result.Amount = amount
		}
		result.Deposits = delegatorService.GetDeposits(address)
	}

	result.WithdrawAddress = lcd.QueryWithdrawAddr(address)
	result.IsProfiler = isProfiler(address)
	result.Address = address
	return result
}

func (service *AccountService) QueryRichList() interface{} {

	result, err := document.Account{}.GetAccountList()

	if err != nil {
		logger.Error("GetAccountList have error", logger.String("err", err.Error()))
		panic(types.CodeNotFound)
	}

	var accList []vo.AccountInfo
	var totalAmt = float64(0)

	for _, acc := range result {
		totalAmt += acc.Total.Amount
	}

	for index, acc := range result {
		rate, _ := utils.NewRatFromFloat64(acc.Total.Amount / totalAmt).Float64()
		accList = append(accList, vo.AccountInfo{
			Rank:    index + 1,
			Address: acc.Address,
			Balance: utils.Coins{
				acc.Total,
			},
			Percent:  rate,
			UpdateAt: acc.TotalUpdateAt,
		})
	}
	return accList
}

func isProfiler(address string) bool {
	genesis := commonService.GetGenesis()
	for _, profiler := range genesis.Result.Genesis.AppState.Guardian.Profilers {
		if profiler.Address == address {
			return true
		}
	}
	return false
}

func (service *AccountService) QueryDelegations(address string) (result []*vo.AccountDelegationsVo) {
	delegations := lcd.GetDelegationsByDelAddr(address)
	result = make([]*vo.AccountDelegationsVo, 0, len(delegations))
	for _, val := range delegations {
		data := vo.AccountDelegationsVo{
			Address: val.ValidatorAddr,
			Shares:  val.Shares,
			Height:  val.Height,
		}
		valdator, err := document.Validator{}.QueryValidatorDetailByOperatorAddr(val.ValidatorAddr)
		if err == nil {
			data.Moniker = valdator.Description.Moniker
			data.Amount = computeVotingPower(valdator, val.Shares)
		}
		result = append(result, &data)
	}

	return result
}

func computeVotingPower(validator document.Validator, shares string) utils.Coin {
	rate, err := utils.QuoByStr(validator.Tokens, validator.DelegatorShares)
	if err != nil {
		logger.Error("validator.Tokens / validator.DelegatorShares", logger.String("err", err.Error()))
		rate, _ = new(big.Rat).SetString("1")
	}
	sharesAsRat, ok := new(big.Rat).SetString(shares)
	if !ok {
		logger.Error("convert validator.Tokens type (string to big.Rat) ", logger.Any("result", ok),
			logger.String("validator tokens", validator.Tokens))
	}

	tokensAsRat := new(big.Rat)
	tokensAsRat.Mul(rate, sharesAsRat)
	amount, _ := strconv.ParseFloat(tokensAsRat.FloatString(4), 64)

	return utils.Coin{
		Amount: amount,
		Denom:  constant.IrisAttoUnit,
	}
}

func (service *AccountService) QueryUnbondingDelegations(address string) (result []*vo.AccountUnbondingDelegationsVo) {

	unbondingdelegations := lcd.GetUnbondingDelegationsByDelegatorAddr(address)

	for _, val := range unbondingdelegations {
		data := vo.AccountUnbondingDelegationsVo{
			Address: val.ValidatorAddr,
			EndTime: val.MinTime,
			Height:  val.CreationHeight,
			Amount:  utils.ParseCoin(val.Balance),
		}
		valdator, err := document.Validator{}.QueryValidatorDetailByOperatorAddr(val.ValidatorAddr)
		if err == nil {
			data.Moniker = valdator.Description.Moniker
		}

	}
	return result
}

func (service *AccountService) QueryRewards(address string) (result vo.AccountRewardsVo) {

	commissionrewards, delegationrewards, rewards, err := lcd.GetDistributionRewardsByValidatorAcc(address)
	if err != nil {
		logger.Error("GetDistributionRewardsByValidatorAcc have error", logger.String("err", err.Error()))
		return
	}

	result.CommissionRewards = commissionrewards
	result.TotalRewards = rewards
	for _, val := range delegationrewards {
		data := vo.DelagationsRewards{Address: val.Validator, Amount: val.Reward}
		valdator, err := document.Validator{}.QueryValidatorDetailByOperatorAddr(val.Validator)
		if err == nil {
			data.Moniker = valdator.Description.Moniker
		}
		result.DelagationsRewards = append(result.DelagationsRewards, data)
	}

	return result
}
