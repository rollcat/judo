package libjudo

func assert(err error) {
	if err != nil {
		panic(err)
	}
}
