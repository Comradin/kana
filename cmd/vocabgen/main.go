// Command vocabgen builds vocab/data/words.json.gz from JMdict and the JLPT
// lists. Run by hand; inputs are downloaded by the maintainer.
//
//	go run ./cmd/vocabgen -jmdict JMdict.gz -jlpt yomitan-jlpt-vocab/original_data -out vocab/data/words.json.gz
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	jmdict := flag.String("jmdict", "", "path to JMdict or JMdict.gz (full multilingual)")
	jlpt := flag.String("jlpt", "", "directory with n1.csv … n5.csv (yomitan-jlpt-vocab original_data)")
	out := flag.String("out", "vocab/data/words.json.gz", "output file")
	flag.Parse()
	if *jmdict == "" || *jlpt == "" {
		flag.Usage()
		os.Exit(2)
	}
	tmp := *out + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	st, err := run(*jmdict, *jlpt, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Rename(tmp, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(st)
}
