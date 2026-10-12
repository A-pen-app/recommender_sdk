package model

import "time"

// Options are the tunable constants of the hot-feed score. Every field keeps
// the value the formula has used so far when left at its zero value, so
// Recommend(ctx, posts, uid) with no options ranks exactly as before. Callers
// (the three API services) pass overrides read from their own env so the
// numbers can be A/B tested without a new SDK tag each time.
type Options struct {
	// UpvoteDivisor divides the upvote count before it is added to the score.
	// The historical value is 2 (integer division, so a single upvote scores 0).
	UpvoteDivisor int
	// NonAnonymousFactor multiplies the score of a non-anonymous item.
	// Historical value 2.
	NonAnonymousFactor float64
	// FemaleFactor multiplies the score of an item whose author is female. It
	// replaces (does not stack on) NonAnonymousFactor, matching the historical
	// behaviour. Historical value 4.
	FemaleFactor float64

	now time.Time
}

// Option mutates Options; build them with the With* constructors.
type Option func(*Options)

const (
	defaultUpvoteDivisor      = 2
	defaultNonAnonymousFactor = 2.
	defaultFemaleFactor       = 4.
)

// DefaultOptions returns the constants the formula used before they became
// configurable.
func DefaultOptions() Options {
	return Options{
		UpvoteDivisor:      defaultUpvoteDivisor,
		NonAnonymousFactor: defaultNonAnonymousFactor,
		FemaleFactor:       defaultFemaleFactor,
	}
}

// ApplyOptions folds opts onto the defaults. Zero/negative numeric overrides
// are ignored so a missing env var never turns a divisor into 0.
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

// Now is the single timestamp every score in one Recommend call is aged
// against, so the ordering cannot drift while the sort runs.
func (o Options) Now() time.Time { return o.now }

func WithUpvoteDivisor(n int) Option { return func(o *Options) { o.UpvoteDivisor = n } }

func WithNonAnonymousFactor(f float64) Option {
	return func(o *Options) { o.NonAnonymousFactor = f }
}

func WithFemaleFactor(f float64) Option { return func(o *Options) { o.FemaleFactor = f } }

// WithNow pins the reference time; tests use it, callers normally do not.
func WithNow(t time.Time) Option { return func(o *Options) { o.now = t } }
