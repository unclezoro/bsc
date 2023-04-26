package vote

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/v3/io/file"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

const (
	voteSignerTimeout = time.Second * 5
)

type VoteSigner struct {
	km     *keymanager.IKeymanager
	pubKey [48]byte
}

func IsValid(walletDir string) (bool, error) {
	expanded, err := file.ExpandPath(walletDir)
	if err != nil {
		return false, err
	}
	f, err := os.Open(expanded) // #nosec G304
	if err != nil {
		if strings.Contains(err.Error(), "no such file") ||
			strings.Contains(err.Error(), "cannot find the file") ||
			strings.Contains(err.Error(), "cannot find the path") {
			panic(fmt.Sprintf("can not open %s, %v", expanded, err))
			return false, nil
		}
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
		}
	}()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return false, err
	}

	if len(names) == 0 {
		panic(fmt.Sprintf("get zero wallet types in %s", expanded))
		return false, fmt.Errorf("get zero wallet types in %s", expanded)
	}

	fmt.Printf("names, len: %v, %d\n", names, len(names))

	// Count how many wallet types we have in the directory
	numWalletTypes := 0
	for _, name := range names {
		// Nil error means input name is `derived`, `remote` or `imported`
		_, err = keymanager.ParseKind(name)
		if err == nil {
			numWalletTypes++
		}
	}
	fmt.Println("numWalletTypes: ", numWalletTypes)
	return numWalletTypes == 1, nil
}

func NewVoteSigner(blsPasswordPath, blsWalletPath string) (*VoteSigner, error) {
	IsValid(blsWalletPath)
	dirExists, err := wallet.Exists(blsWalletPath)
	if err != nil {
		log.Error("Check BLS wallet exists", "err", err)
		return nil, err
	}
	if !dirExists {
		log.Error("BLS wallet did not exists.")
		return nil, fmt.Errorf("BLS wallet did not exists.")
	}

	walletPassword, err := ioutil.ReadFile(blsPasswordPath)
	if err != nil {
		log.Error("Read BLS wallet password", "err", err)
		return nil, err
	}
	log.Info("Read BLS wallet password successfully")

	w, err := wallet.OpenWallet(context.Background(), &wallet.Config{
		WalletDir:      blsWalletPath,
		WalletPassword: string(walletPassword),
	})
	if err != nil {
		log.Error("Open BLS wallet failed", "err", err)
		return nil, err
	}
	log.Info("Open BLS wallet successfully")

	km, err := w.InitializeKeymanager(context.Background(), iface.InitKeymanagerConfig{ListenForChanges: false})
	if err != nil {
		log.Error("Initialize key manager failed", "err", err)
		return nil, err
	}
	log.Info("Initialized keymanager successfully")

	ctx, cancel := context.WithTimeout(context.Background(), voteSignerTimeout)
	defer cancel()

	pubKeys, err := km.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch validating public keys")
	}

	return &VoteSigner{
		km:     &km,
		pubKey: pubKeys[0],
	}, nil
}

func (signer *VoteSigner) SignVote(vote *types.VoteEnvelope) error {
	// Sign the vote, fetch the first pubKey as validator's bls public key.
	pubKey := signer.pubKey
	blsPubKey, err := bls.PublicKeyFromBytes(pubKey[:])
	if err != nil {
		return errors.Wrap(err, "convert public key from bytes to bls failed")
	}

	voteDataHash := vote.Data.Hash()

	ctx, cancel := context.WithTimeout(context.Background(), voteSignerTimeout)
	defer cancel()

	signature, err := (*signer.km).Sign(ctx, &validatorpb.SignRequest{
		PublicKey:   pubKey[:],
		SigningRoot: voteDataHash[:],
	})
	if err != nil {
		return err
	}

	copy(vote.VoteAddress[:], blsPubKey.Marshal()[:])
	copy(vote.Signature[:], signature.Marshal()[:])
	return nil
}

// Metrics to indicate if there's any failed signing.
func votesSigningErrorMetric(blockNumber uint64, blockHash common.Hash) metrics.Gauge {
	return metrics.GetOrRegisterGauge(fmt.Sprintf("voteSigning/blockNumber/%d/blockHash/%s", blockNumber, blockHash), nil)
}
