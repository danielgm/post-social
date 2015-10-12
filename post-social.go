package main

/*
Takes a folder containing frames and outputs:
- a GIF animation less than 540x800 and less than 2.0 MB for Tumblr.
- an h264 video less than 1280x for Facebook.
- a GIF animation less than 1024x512 and less than 3.0 MB for Twitter.
- a video at 510x510 and from 3-15 seconds for Instagram.
*/

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	inputDir  string
	outputDir string
	tempDir   string
)

func main() {
	var err error

	inputDir = os.Args[1]

	outputDir = inputDir + "/post-social"
	if _, err := os.Stat(outputDir); err == nil {
		rm(outputDir)
	}
	os.Mkdir(outputDir, 0755)

	tempDir = outputDir + "/temp"
	os.Mkdir(tempDir, 0755)

	contents, err := ioutil.ReadDir(inputDir)
	check(err)

	fileType := getFileType(contents)
	fmt.Println(fileType)
	generateGif([]string{"540x540", "405x405", "270x270"}, 2, "tumblr", fileType)
	generateGif([]string{"640x640", "480x480", "320x320"}, 3, "twitter", fileType)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func runArgs(command string, args ...string) {
	run(command, args)
}

func run(command string, args []string) {
	cmd := exec.Command(command, args...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if stdout.Len() > 0 {
		fmt.Printf("STDOUT: %q\n", stdout.String())
	}
	if stderr.Len() > 0 {
		fmt.Printf("STDERR: %q\n", stderr.String())
		panic(stderr.String())
	}

	check(err)
}

func cp(from string, to string) {
	fmt.Println("%> cp " + from + " " + to)
	fromGlob, err := filepath.Glob(from)
	check(err)

	to, err = filepath.Abs(to)
	check(err)

	args := append(fromGlob, to)
	run("cp", args)
}

func rm(path string) {
	fmt.Println("%> rm " + path)

	pathGlob, err := filepath.Glob(path)
	check(err)

	args := append([]string{"-rf"}, pathGlob...)
	run("rm", args)
}

func mogrify(dimensionArg, path string) {
	pathGlob, err := filepath.Glob(path)
	check(err)
	args := []string{"-resize", dimensionArg + "^", "-format", "gif"}

	fmt.Println("%> mogrify " + strings.Join(args, " ") + " " + path)

	args = append(args, pathGlob...)
	run("mogrify", args)
}

func gifsicle(delay int, colors int, inputPath string, outputPath string) {
	pathGlob, err := filepath.Glob(inputPath)
	check(err)

	args := []string{
		fmt.Sprintf("--delay=%d", delay),
		"--loop",
		fmt.Sprintf("--colors=%d", colors)}

	fmt.Println("%> gifsicle " + strings.Join(args, " ") + " " + inputPath)

	args = append(args, pathGlob...)

	outputFile, err := os.Create(outputPath)
	check(err)
	defer outputFile.Close()

	cmd := exec.Command("gifsicle", args...)
	cmd.Stdout = outputFile
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Start()
	check(err)
	cmd.Wait()

	if stderr.Len() > 0 {
		fmt.Printf("STDERR: %q\n", stderr.String())
		panic(stderr.String())
	}

	check(err)
}

func duk(path string) int {
	fmt.Println("%> du -sh " + path)

	cmd := exec.Command("du", "-k", path)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if stdout.Len() > 0 {
		fmt.Print(stdout.String())
		size, err := strconv.Atoi(strings.Split(stdout.String(), "\t")[0])
		check(err)

		return size
	}

	if stderr.Len() > 0 {
		fmt.Printf("STDERR: %q\n", stderr.String())
		panic(stderr.String())
	}

	check(err)

	panic("No output from du.")
}

func getFileType(files []os.FileInfo) string {
	re := regexp.MustCompile("^frame\\d{4}\\.(bmp|gif|jpg|png)$")
	for _, fileInfo := range files {
		if re.MatchString(filepath.Base(fileInfo.Name())) {
			return filepath.Ext(fileInfo.Name())[1:]
		}
	}
	return ""
}

func getImages(files []os.FileInfo, fileType string) []os.FileInfo {
	re := regexp.MustCompile("^frame\\d{4}\\." + fileType + "$")
	var result []os.FileInfo
	for _, fileInfo := range files {
		if re.MatchString(filepath.Base(fileInfo.Name())) {
			result = append(result, fileInfo)
		}
	}
	return result
}

func generateGif(dimensionAttempts []string, sizeLimitMb int, target string, fileType string) {
	success := false
	for i := 0; i < len(dimensionAttempts); i++ {
		rm(tempDir + "/*")
		cp(inputDir+"/frame*."+fileType, tempDir)

		dimensions := dimensionAttempts[i]
		mogrify(dimensions, tempDir+"/frame*."+fileType)

		for colors := 256; colors > 16; colors -= 32 {
			outputFilepath := outputDir + "/" + target + dimensions + "-" + strconv.Itoa(colors) + ".gif"

			gifsicle(3, colors, tempDir+"/frame*.gif", outputFilepath)

			kb := duk(outputFilepath)
			if kb < sizeLimitMb*1024 {
				success = true
				break
			}
		}
	}

	if !success {
		fmt.Println("WARN: Failed to generate small enough file for " + target + ".")
	}

	rm(tempDir + "/*")
}
