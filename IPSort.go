package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

type ipSlice []net.IP

func (ips ipSlice) Len() int {
	return len(ips)
}

func (ips ipSlice) Less(i, j int) bool {
	return bytes.Compare(ips[i], ips[j]) < 0
}

func (ips ipSlice) Swap(i, j int) {
	ips[i], ips[j] = ips[j], ips[i]
}

func sortIPsFromFile(filePath string) ([]net.IP, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Unable to open file(%s):error(%s)\n", file.Name(), err)
		}
	}(file)

	var ips ipSlice
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ipConfig := scanner.Text()
		ipString := strings.Split(ipConfig, ":")
		ip := net.ParseIP(ipString[0])
		if ip != nil {
			ips = append(ips, ip)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Sort(ips)
	return ips, nil
}

func removeIPsFromSet(filePath string, removeIPs []net.IP) ([]net.IP, error) {
	ips, err := sortIPsFromFile(filePath)
	if err != nil {
		return nil, err
	}

	removeMap := make(map[string]struct{})
	for _, ip := range removeIPs {
		removeMap[ip.String()] = struct{}{}
	}

	var result []net.IP
	for _, ip := range ips {
		if _, found := removeMap[ip.String()]; !found {
			result = append(result, ip)
		}
	}

	return result, nil
}

func hasIPDuplicated(slice []net.IP) (bool, net.IP) {
	elementMap := make(map[int]struct{})

	for _, element := range slice {
		ip := element.To4()
		if ip == nil {
			continue
		}
		var ipInt int = int(ip[0])<<24 + int(ip[1])<<16 + int(ip[2])<<8 + int(ip[3])
		if _, exists := elementMap[ipInt]; exists {
			return true, element
		}
		elementMap[ipInt] = struct{}{}
	}
	return false, nil
}

func writeIPsToFile(ips []net.IP, flags int, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Unable to close file(%s):error(%s)\n", file.Name(), err)
		}
	}(file)

	writer := bufio.NewWriter(file)
	for _, ip := range ips {
		var cmd string

		if flags == 1 {
			// we only use the sorted IPs
			cmd = ip.String()
		} else {
			cmd = "response=$(curl -m 3 -x http://$user:$pass@" + ip.String() + ":$port $URL --silent " +
				"--write-out \"%{http_code}\" --output /dev/null)\n"
			cmd += "if [ \"$response\" -eq \"000\" ]; then\n"
			cmd += "	if [ $error_occurred -eq 0 ]; then\n"
			cmd += "		echo \"\"\n"
			cmd += "		error_occurred=1\n"
			cmd += "	fi\n"
			cmd += "	echo \"Error:" + ip.String() + "\"\n"
			cmd += "else\n"
			cmd += "	echo -n \"*\"\n"
			cmd += "	error_occurred=0\n"
			cmd += "fi\n"
			cmd += "sleep 0.01\n"
		}
		fmt.Println(ip.String())
		_, err := fmt.Fprintln(writer, cmd)
		if err != nil {
			break
		}
	}
	return writer.Flush()
}

func main() {
	inputFile := flag.String("i", "", "Input IPs file")
	fmtFlagStr := flag.String("f", "", "Format flag (0/1)")
	removeFile := flag.String("r", "", "Remove IPs file")
	outputFile := flag.String("o", "", "Output IPs file")

	flag.Parse()

	if *inputFile == "" || *fmtFlagStr == "" || *outputFile == "" {
		fmt.Println("Usage:", os.Args[0], "-i input-file -f 0/1 -o output-file")
		return
	}

	fmtFlag, err := strconv.Atoi(*fmtFlagStr)
	if err != nil {
		fmt.Println("Error: format flag must be an integer (0/1)")
		return
	}

	var ips []net.IP
	if *removeFile != "" {
		removeIPs, err := sortIPsFromFile(*removeFile)
		if err != nil {
			fmt.Println("Error:", err)
		}
		ips, err = removeIPsFromSet(*inputFile, removeIPs)
		if err != nil {
			fmt.Println("Error:", err)
		}
	} else {
		ips, err = sortIPsFromFile(*inputFile)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	isDuplicated, ip := hasIPDuplicated(ips)
	if isDuplicated {
		fmt.Printf("Error: IP(%s) address is duplicated\n", ip.String())
		return
	}

	err = writeIPsToFile(ips, fmtFlag, *outputFile)
	if err != nil {
		fmt.Println("Error writing sorted IPs:", err)
		return
	}

	fmt.Println("IPs sorted and written to", *outputFile)
}
