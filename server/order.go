package server

import "strings"

func orderNode(list []*node) {
	f := func(a, b *node) bool {
		if strings.Contains(a.addr, "(localhost)") {
			return true
		} else if strings.Contains(b.addr, "(localhost)") {
			return false
		}
		return a.uuid < b.uuid
	}
	max_len := len(list)
	tmp := make([]*node, max_len)
	for i := 0; i < max_len-max_len&1; i += 2 {
		if f(list[i+1], list[i]) {
			list[i], list[i+1] = list[i+1], list[i]
		}

	}
	for i := 0; i < max_len-max_len&3; i += 4 {
		if f(list[i+2], list[i]) {
			list[i], list[i+2] = list[i+2], list[i]
		}
		if f(list[i+3], list[i+1]) {
			list[i+1], list[i+3] = list[i+3], list[i+1]
		}
		if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}

	}
	if max_len&3 == 3 {
		i := max_len - 3
		if f(list[i+2], list[i]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
			list[i], list[i+1] = list[i+1], list[i]
		} else if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}
	}
	var step, l, max, r int
	step = 4
	for step < max_len {
		step <<= 1
		for i := 0; i < max_len; i += step {
			l, r, max = i, i+step/2, i+step
			if max > max_len {
				max = max_len
			}
			for index := i; index < max; index++ {
				if l == step/2+i || (r < max && f(list[r], list[l])) {
					tmp[index] = list[r]
					r++
				} else {
					tmp[index] = list[l]
					l++
				}
			}
		}
		if step < max_len {
			for i := 0; i < max_len; i += step {
				l, r, max = i, i+step/2, i+step
				if max > max_len {
					max = max_len
				}
				for index := i; index < max; index++ {
					if l == step/2+i || (r < max && f(tmp[r], tmp[l])) {
						list[index] = tmp[r]
						r++
					} else {
						list[index] = tmp[l]
						l++
					}
				}
			}
		} else {
			copy(list, tmp)
		}
	}
}
func orderClientListen(list []*clientListen) {
	f := func(a, b *clientListen) bool {
		return a.id < b.id
	}
	max_len := len(list)
	tmp := make([]*clientListen, max_len)
	for i := 0; i < max_len-max_len&1; i += 2 {
		if f(list[i+1], list[i]) {
			list[i], list[i+1] = list[i+1], list[i]
		}

	}
	for i := 0; i < max_len-max_len&3; i += 4 {
		if f(list[i+2], list[i]) {
			list[i], list[i+2] = list[i+2], list[i]
		}
		if f(list[i+3], list[i+1]) {
			list[i+1], list[i+3] = list[i+3], list[i+1]
		}
		if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}

	}
	if max_len&3 == 3 {
		i := max_len - 3
		if f(list[i+2], list[i]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
			list[i], list[i+1] = list[i+1], list[i]
		} else if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}
	}
	var step, l, max, r int
	step = 4
	for step < max_len {
		step <<= 1
		for i := 0; i < max_len; i += step {
			l, r, max = i, i+step/2, i+step
			if max > max_len {
				max = max_len
			}
			for index := i; index < max; index++ {
				if l == step/2+i || (r < max && f(list[r], list[l])) {
					tmp[index] = list[r]
					r++
				} else {
					tmp[index] = list[l]
					l++
				}
			}
		}
		if step < max_len {
			for i := 0; i < max_len; i += step {
				l, r, max = i, i+step/2, i+step
				if max > max_len {
					max = max_len
				}
				for index := i; index < max; index++ {
					if l == step/2+i || (r < max && f(tmp[r], tmp[l])) {
						list[index] = tmp[r]
						r++
					} else {
						list[index] = tmp[l]
						l++
					}
				}
			}
		} else {
			copy(list, tmp)
		}
	}
}
func orderHttpProxy(list []*httpProxyClient) {
	f := func(a, b *httpProxyClient) bool {
		return a.id < b.id
	}
	max_len := len(list)
	tmp := make([]*httpProxyClient, max_len)
	for i := 0; i < max_len-max_len&1; i += 2 {
		if f(list[i+1], list[i]) {
			list[i], list[i+1] = list[i+1], list[i]
		}

	}
	for i := 0; i < max_len-max_len&3; i += 4 {
		if f(list[i+2], list[i]) {
			list[i], list[i+2] = list[i+2], list[i]
		}
		if f(list[i+3], list[i+1]) {
			list[i+1], list[i+3] = list[i+3], list[i+1]
		}
		if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}

	}
	if max_len&3 == 3 {
		i := max_len - 3
		if f(list[i+2], list[i]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
			list[i], list[i+1] = list[i+1], list[i]
		} else if f(list[i+2], list[i+1]) {
			list[i+1], list[i+2] = list[i+2], list[i+1]
		}
	}
	var step, l, max, r int
	step = 4
	for step < max_len {
		step <<= 1
		for i := 0; i < max_len; i += step {
			l, r, max = i, i+step/2, i+step
			if max > max_len {
				max = max_len
			}
			for index := i; index < max; index++ {
				if l == step/2+i || (r < max && f(list[r], list[l])) {
					tmp[index] = list[r]
					r++
				} else {
					tmp[index] = list[l]
					l++
				}
			}
		}
		if step < max_len {
			for i := 0; i < max_len; i += step {
				l, r, max = i, i+step/2, i+step
				if max > max_len {
					max = max_len
				}
				for index := i; index < max; index++ {
					if l == step/2+i || (r < max && f(tmp[r], tmp[l])) {
						list[index] = tmp[r]
						r++
					} else {
						list[index] = tmp[l]
						l++
					}
				}
			}
		} else {
			copy(list, tmp)
		}
	}
}
