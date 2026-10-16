package model

import "time"

type Options struct {
	UpvoteDivisor      int
	NonAnonymousFactor float64
	FemaleFactor       float64

	now time.Time
}

type Option func(*Options)

const (
	defaultUpvoteDivisor      = 2
	defaultNonAnonymousFactor = 2.
	defaultFemaleFactor       = 4.
)

func DefaultOptions() Options {
	return Options{
		UpvoteDivisor:      defaultUpvoteDivisor,
		NonAnonymousFactor: defaultNonAnonymousFactor,
		FemaleFactor:       defaultFemaleFactor,
	}
}

func ApplyOptions(opts ...Option) Options {
	o := DefaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	if o.UpvoteDivisor <= 0 {
		o.UpvoteDivisor = defaultUpvoteDivisor
	}
	if o.NonAnonymousFactor <= 0 {
		o.NonAnonymousFactor = defaultNonAnonymousFactor
	}
	if o.FemaleFactor <= 0 {
		o.FemaleFactor = defaultFemaleFactor
	}
	if o.now.IsZero() {
		o.now = time.Now()
	}
	return o
}

func (o Options) Now() time.Time { return o.now }

func WithUpvoteDivisor(n int) Option { return func(o *Options) { o.UpvoteDivisor = n } }

func WithNonAnonymousFactor(f float64) Option {
	return func(o *Options) { o.NonAnonymousFactor = f }
}

func WithFemaleFactor(f float64) Option { return func(o *Options) { o.FemaleFactor = f } }

func WithNow(t time.Time) Option { return func(o *Options) { o.now = t } }
