package healthcheck

import (
	context "context"
	"strings"
	sync "sync"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

type Service struct {
	ds  []HealthZ
	log *log.Helper
}

func NewService(logger log.Logger, ds ...HealthZ) *Service {
	return &Service{ds: ds, log: log.NewHelper(logger)}
}

func (s *Service) HealthzCheck(ctx context.Context, _ *HealthzCheckRequest) (*HealthzCheckReply, error) {

	if s.ds == nil {

		return &HealthzCheckReply{Success: true}, nil
	}

	// avoid overhead(invoke goroutine) if only have one datasource that what to ping
	if len(s.ds) == 1 {
		if err := s.ds[0].Ping(ctx); err != nil {
			return nil, errors.InternalServer("internal", err.Error())
		}
		return &HealthzCheckReply{Success: true}, nil
	}

	errsCH := make(chan error, len(s.ds))
	wg := sync.WaitGroup{}

	defer close(errsCH)

	for _, z := range s.ds {
		z := z
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := z.Ping(ctx); err != nil {
				errsCH <- err
			}
		}()
	}

	wg.Wait()

	var errrs []error

	for range s.ds {
		select {
		case e, ok := <-errsCH:
			if !ok {
				break
			}
			errrs = append(errrs, e)
		default:
			// no data in channel
		}
	}

	if len(errrs) == 0 {
		return &HealthzCheckReply{Success: true}, nil
	}

	errStrs := make([]string, len(errrs))

	for i, v := range errrs {
		errStrs[i] = v.Error()
	}

	return nil, errors.InternalServer("internal", strings.Join(errStrs, ","))
}

func (s *Service) PingCheck(ctx context.Context, _ *PingCheckRequest) (*PingCheckReply, error) {
	return &PingCheckReply{Success: true}, nil
}
