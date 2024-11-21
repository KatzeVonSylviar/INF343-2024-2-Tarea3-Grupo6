package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Estructura para representar un paralelo y su token
type Parallel struct {
	TokenHolder int        // Proceso que tiene el token
	RequestQ    []int      // Cola de solicitudes
	Capacity    int        // Capacidad máxima
	Current     int        // Número actual de estudiantes inscritos
	Mutex       sync.Mutex // Mutex para sincronizar acceso
}

// Estructura para representar un proceso
type Process struct {
	ID         int
	Parallels  map[string]*Parallel
	InputFile  string
	OutputFile string
	Mutex      *sync.Mutex
	Channels   map[int]chan string // Canales para comunicación entre procesos
}

/////////////////////////////
// Función Main
/////////////////////////////

func main() {
	// Verificar argumentos de entrada
	if len(os.Args) != 2 {
		fmt.Println("Error en al iniciar, recuerde que se debe iniciar el programa con el siguiente comando: go run mainB.go <Numero_de_Goroutines>")
		return
	}

	NumeroDeGoroutines, err := strconv.Atoi(os.Args[1])
	if err != nil || NumeroDeGoroutines <= 0 {
		fmt.Println("El número de Goroutines debe ser un entero positivo.")
		return
	}

	// Configuración inicial
	InputFile := "solicitudes.txt"
	Paralelos := CargarDatosDeParalelos("paralelos.txt")
	OutputFile := "inscritos.txt"

	// Limpiar el archivo de salida para facilitar pruebas
	os.WriteFile(OutputFile, []byte{}, 0644)

	var mutex sync.Mutex
	//Se generan canales para las Goroutines
	Canales := make(map[int]chan string)
	for i := 1; i <= NumeroDeGoroutines; i++ {
		Canales[i] = make(chan string, 10)
	}

	var wg sync.WaitGroup

	// Crear y lanzar procesos (Goroutines)
	for i := 1; i <= NumeroDeGoroutines; i++ {
		wg.Add(1)
		process := &Process{
			ID:         i,
			Parallels:  Paralelos,
			InputFile:  InputFile,
			OutputFile: OutputFile,
			Mutex:      &mutex,
			Channels:   Canales,
		}
		go process.Goroutine(&wg)
	}

	wg.Wait()
	//Print para verificar final del proceso
	fmt.Println("Simulación completa. Revisa el archivo de salida.")
}

// ///////////////////////////
// Función para de cada Goroutine
// ///////////////////////////

func (p *Process) Goroutine(wg *sync.WaitGroup) {
	defer wg.Done()

	c := 0

	for {

		// Se lee una solicitud del archivo "solicitudes.txt"
		p.Mutex.Lock()
		SolicitudDelEstudiante := p.LeerSolicitud()
		p.Mutex.Unlock()

		// Si no queda solicitudes de estudiantes, revisar solicitudes por un tiempo extra antes de terminar.
		if SolicitudDelEstudiante == "" {
			// No hay más solicitudes
			//fmt.Printf("Proceso %d no encontro más solicitudes, revisando solicitudes de Token\n", p.ID)
			for paralelo := range p.Parallels {
				//fmt.Printf("Proceso %d esta revisando consultas del paralelo %s\n", p.ID, paralelo)
				if p.ComprobarSolicitudesDeTokens(paralelo) {
					c = 0
				}
			}
			c++
			if c >= 10 {
				return
			}
			continue
		}

		// Se extraen los datos necesarios de la solicitud
		SeccionesDeSolicitud := strings.Fields(SolicitudDelEstudiante)
		if len(SeccionesDeSolicitud) < 2 {
			fmt.Printf("Proceso %d encontró una fila inválida: %s\n", p.ID, SolicitudDelEstudiante)
			continue
		}

		NombreDelEstudiante := SeccionesDeSolicitud[0]
		Preferencias := SeccionesDeSolicitud[1:]

		// Intentar inscribir al estudiante
		for _, paralelo := range Preferencias {
			//fmt.Printf("Proceso %d inscribira en el paralelo %s\n", p.ID, paralelo)
			if p.RealizarInscripcion(NombreDelEstudiante, paralelo) {
				break
			}
		}

		//Se lleva a cabo una espera antes de revisar las solicitudes de Tokens
		//Esto con el fin de prevenir deadlocks

		// Revisar solicitudes de Tokens (Se aprovechara el hecho de que en las preferencias se tiene la lista completa de paralelos)
		for _, paralelo := range Preferencias {
			//fmt.Printf("Proceso %d esta revisando consultas del paralelo %s\n", p.ID, paralelo)
			p.ComprobarSolicitudesDeTokens(paralelo)
		}
	}
}

/////////////////////////////
// Funciónes para simular el algoritmo de Susuki-Kasami
/////////////////////////////

// Solicitar el token si no se tiene y luego inscribir al estudiante
func (p *Process) RealizarInscripcion(NombreDelEstudiante, paralelo string) bool {

	// Obtener paralelo de la solicitud
	par := p.Parallels[paralelo]

	par.Mutex.Lock()
	// Si no tengo el token asignado, lo solicito.
	if par.TokenHolder != p.ID {
		par.RequestQ = append(par.RequestQ, p.ID)
		//fmt.Printf("Proceso %d solicitó token para %s\n", p.ID, paralelo)
		par.Mutex.Unlock()

		// Esperar que el Token sea entregado
		for {
			msg := <-p.Channels[p.ID]
			if msg == fmt.Sprintf("token:%s", paralelo) {
				break
			}
		}
	} else {
		// Si tengo el token asignado, no necesario solicitarlo
		par.Mutex.Unlock()
	}

	// Se realiza el proceso de inscribir al estudiante si hay cupos
	if par.Current < par.Capacity {
		par.Current++
		p.NotificarInscripcion(NombreDelEstudiante, paralelo)

		return true
	}

	// Si no se realizo el proceso se notifica para que la Goroutine intente con el siguiente paralelo
	return false
}

// Función adicional para manejar solicitudes
func (p *Process) ComprobarSolicitudesDeTokens(paralelo string) bool {
	par := p.Parallels[paralelo]

	//Se verifica si hay solicitudes para el token si se es dueño del token
	if len(par.RequestQ) > 0 && par.TokenHolder == p.ID {
		SiguienteDueño := par.RequestQ[0]
		par.RequestQ = par.RequestQ[1:]
		par.TokenHolder = SiguienteDueño
		//fmt.Printf("Proceso %d transfiere token a P%d para %s\n", p.ID, SiguienteDueño, paralelo)
		p.Channels[SiguienteDueño] <- fmt.Sprintf("token:%s", paralelo)
		// Se atendio una solicitud
		return true
	}
	// No existia solicitud
	return false
}

// Reportar inscripción en el archivo de salida
func (p *Process) NotificarInscripcion(NombreDelEstudiante, paralelo string) {
	file, err := os.OpenFile(p.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Proceso %d no pudo escribir en el archivo de salida: %v\n", p.ID, err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	writer.WriteString(fmt.Sprintf("P%d: %s %s\n", p.ID, NombreDelEstudiante, paralelo))
	writer.Flush()

	//fmt.Printf("Proceso %d inscribió a %s en %s\n", p.ID, NombreDelEstudiante, paralelo)
}

/////////////////////////////
// Funciónes para asistir al proceso
/////////////////////////////

// Leer y borrar una línea del un archivo (Utilizacodo para leer las solicitudes)
func (p *Process) LeerSolicitud() string {
	file, err := os.OpenFile(p.InputFile, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Proceso %d no pudo abrir el archivo: %v\n", p.ID, err)
		return ""
	}
	defer file.Close()

	// Leer todas las líneas
	scanner := bufio.NewScanner(file)
	var Lineas []string
	var PrimeraLinea string
	for scanner.Scan() {
		if PrimeraLinea == "" {
			PrimeraLinea = scanner.Text()
		} else {
			Lineas = append(Lineas, scanner.Text())
		}
	}

	// Escribir las líneas (Solicitudes) restantes
	file.Truncate(0)
	file.Seek(0, 0)
	writer := bufio.NewWriter(file)
	for _, line := range Lineas {
		writer.WriteString(line + "\n")
	}
	writer.Flush()

	return PrimeraLinea
}

// Cargar las capacidades de los paralelos desde el archivo "paralelos.txt"
// Tambien "inicializa" los paralelos, entregando todos los tokens al primer proceso existenbte
func CargarDatosDeParalelos(fileName string) map[string]*Parallel {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("Error al abrir el archivo de paralelos: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	Paralelos := make(map[string]*Parallel)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 2 {
			continue
		}
		capacity, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		// Inicialmente el proceso 1 tiene todos los tokens
		Paralelos[parts[0]] = &Parallel{
			TokenHolder: 1,
			Capacity:    capacity,
			RequestQ:    []int{},
		}
	}

	return Paralelos
}
