package otpsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"doorman/internal/domain"
	otphandler "doorman/internal/handler/otp"
	tokenhandler "doorman/internal/handler/token"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"
	"time"
)

const (
	otpTTL         = 3 * time.Minute
	regTokenTTL    = 15 * time.Minute
	maxOTPAttempts = 3
	regTokenLength = 32
)

type Service struct {
	identityStore IdentityStore
	regTokenStore RegTokenStore
	otpStore      OTPStore
	taskScheduler TaskScheduler
	jwtPublisher  JWTPublisher
}

type OtpData struct {
	Hash     string `redis:"hash"`
	Attempts int    `redis:"attempts"`
}

func NewService(
	identityStore IdentityStore,
	regTokenStore RegTokenStore,
	otpStore OTPStore,
	taskScheduler TaskScheduler,
	jwtPublisher JWTPublisher,
) *Service {
	return &Service{
		identityStore: identityStore,
		regTokenStore: regTokenStore,
		taskScheduler: taskScheduler,
		otpStore:      otpStore,
		jwtPublisher:  jwtPublisher,
	}
}

func (s *Service) SendOTP(ctx context.Context, phone string, channel domain.Channel) error {
	exists, err := s.otpStore.Exists(ctx, phone)
	if err != nil {
		return err
	}

	if exists {
		return ErrAlreadySent
	}

	identity, err := s.identityStore.GetByPhone(ctx, phone)

	if err != nil {
		return err
	}

	if identity != nil &&
		(identity.Status == domain.IdentityStatusBlocked || identity.Status == domain.IdentityStatusDeleted) {
		return ErrPhoneUnavailable
	}

	otp, err := generateOTP()
	if err != nil {
		return err
	}

	hashOTP := s.hashOTP(otp)

	err = s.otpStore.Save(ctx, phone, hashOTP, otpTTL)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(struct {
		Phone   string         `json:"phone"`
		OTP     uint64         `json:"otp"`
		Channel domain.Channel `json:"channel"`
	}{
		Phone:   phone,
		OTP:     otp,
		Channel: channel,
	})
	if err != nil {
		return err
	}

	return s.taskScheduler.Schedule(ctx, domain.OTPSent, payload)
}

func (s *Service) VerifyOTP(ctx context.Context, phone string, otp uint64) (otphandler.VerifyResult, error) {
	otpData, err := s.otpStore.Get(ctx, phone)
	if err != nil {
		return nil, err
	}

	if otpData == nil {
		return nil, ErrNotFoundOrExpired
	}

	if otpData.Attempts >= maxOTPAttempts {
		return nil, ErrAttemptsExceeded
	}

	if !s.isValidOTP(otpData.Hash, otp) {
		if err = s.otpStore.IncrAttempts(ctx, phone); err != nil {
			return nil, err
		}
		return nil, ErrInvalid
	}

	if err = s.otpStore.Delete(ctx, phone); err != nil {
		return nil, err
	}

	return s.issueResult(ctx, phone)
}

func (s *Service) issueResult(ctx context.Context, phone string) (otphandler.VerifyResult, error) {
	identity, err := s.identityStore.GetByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	if identity == nil {
		return s.issueRegistrationToken(ctx, phone)
	}

	if identity.Status == domain.IdentityStatusBlocked ||
		identity.Status == domain.IdentityStatusDeleted {
		return nil, ErrPhoneUnavailable
	}

	return s.issueAuthTokens(ctx, identity.ID)
}

func (s *Service) issueAuthTokens(ctx context.Context, userID string) (*tokenhandler.AuthTokens, error) {
	accessToken, refreshToken, expiresIn, err := s.jwtPublisher.IssueTokens(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &tokenhandler.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    uint64(expiresIn),
	}, nil
}

func (s *Service) issueRegistrationToken(ctx context.Context, phone string) (*otphandler.RegistrationToken, error) {
	token, err := generateToken(regTokenLength)
	if err != nil {
		return nil, err
	}

	if err = s.regTokenStore.Save(ctx, phone, token, regTokenTTL); err != nil {
		return nil, err
	}

	return &otphandler.RegistrationToken{
		Token: token,
	}, nil
}

func generateToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateOTP() (uint64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return 0, err
	}
	return n.Uint64() + 100000, nil
}

func (s *Service) hashOTP(otp uint64) string {
	sum := sha256.Sum256([]byte(strconv.FormatUint(otp, 10)))
	return hex.EncodeToString(sum[:])
}

func (s *Service) isValidOTP(storedHash string, otp uint64) bool {
	return storedHash == s.hashOTP(otp)
}
