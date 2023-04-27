package rollingf

type Option interface {
	apply(r *Roll)
}

type optionFunc func(r *Roll)

func (f optionFunc) apply(r *Roll) {
	f(r)
}

// Lock enables/disables rollingf's internal lock when Write. Default the lock is enable.
func Lock(enable bool) Option {
	return optionFunc(func(r *Roll) {
		if !enable {
			r.rwmu = nil
		}
	})
}

// Compress specifies the format of the compressed file
func Compress(format CompressFormat) Option {
	return optionFunc(func(r *Roll) {
		if format == NoCompress {
			return
		}
		r.WithMatcher(CompressMatcher(format)) 
		r.WithProcessor(Compressor(format)) 
	})
}
