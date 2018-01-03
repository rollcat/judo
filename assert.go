package main

func assert(err error) {
	if err != nil {
		panic(err)
	}
}
