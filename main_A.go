package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Paralelo struct {
	Name  string
	Cupos int
	Mux   sync.Mutex // bloqueo por paralelo
}

type Solicitud struct {
	Estudiante string
	Paralelos  []string
}

var (
	paralelos         map[string]*Paralelo
	globalFileMutex   sync.Mutex // para sincronizar acceso a archivos compartidos
	wg                sync.WaitGroup
	processedLogMutex sync.Mutex
)

// leer paralelos desde el archivo
func leerParalelos(filename string) map[string]*Paralelo {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	paralelos := make(map[string]*Paralelo)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) != 2 {
			continue
		}
		cupos, err := strconv.Atoi(line[1])
		if err != nil {
			panic(err)
		}
		paralelos[line[0]] = &Paralelo{Name: line[0], Cupos: cupos}
	}

	return paralelos
}

// leer solicitudes desde el archivo
func leerSolicitudes(filename string) []Solicitud {
	globalFileMutex.Lock()
	defer globalFileMutex.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var solicitudes []Solicitud
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) < 2 {
			continue
		}
		solicitudes = append(solicitudes, Solicitud{Estudiante: line[0], Paralelos: line[1:]})
	}

	return solicitudes
}

// registrar inscripciones en el log
func registrarInscripcion(estudiante, paralelo string) {
	processedLogMutex.Lock()
	defer processedLogMutex.Unlock()

	file, err := os.OpenFile("inscritos.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%s inscrito en %s\n", estudiante, paralelo))
	if err != nil {
		panic(err)
	}
}

// procesar una solicitud
func procesarSolicitud(solicitud Solicitud) {
	defer wg.Done()

	for _, paralelo := range solicitud.Paralelos {
		par, exists := paralelos[paralelo]
		if !exists {
			continue
		}

		par.Mux.Lock() // bloquear solo el paralelo específico
		if par.Cupos > 0 {
			par.Cupos--
			par.Mux.Unlock()
			registrarInscripcion(solicitud.Estudiante, paralelo)
			return
		}
		par.Mux.Unlock()
	}

	registrarInscripcion(solicitud.Estudiante, "No pudo inscribirse en ningún paralelo")
}

// actualizar paralelos en el archivo
func actualizarParalelos(filename string) {
	globalFileMutex.Lock()
	defer globalFileMutex.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for _, paralelo := range paralelos {
		_, err := file.WriteString(fmt.Sprintf("%s %d\n", paralelo.Name, paralelo.Cupos))
		if err != nil {
			panic(err)
		}
	}
}

// funcion main
func main() {
	// leer archivos iniciales
	paralelos = leerParalelos("paralelos.txt")
	solicitudes := leerSolicitudes("solicitudes.txt")

	// procesar solicitudes
	for _, solicitud := range solicitudes {
		wg.Add(1)
		go procesarSolicitud(solicitud)
	}

	wg.Wait() // esperar que todas las goroutines terminen

	// actualizar archivo de paralelos
	actualizarParalelos("paralelos.txt")

	fmt.Println("Procesamiento completado. Ver inscritos.txt y paralelos.txt para resultados.")
}
