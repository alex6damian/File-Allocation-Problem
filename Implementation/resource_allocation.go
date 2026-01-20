package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
)

// |====================================================================|
// |STRUCTURI DE DATE SI FUNCTII PENTRU MODELAREA UNUI SISTEM DISTRIBUIT|
// |====================================================================|

// Nod in sistemul distribuit
type Node struct {
	ID         int
	Lambda     float64 // rata de sosire (cat trafic primeste nodul) 0.2 = 20%
	Mu         float64 // rata de servire (cate cereri/sec poate procesa nodul) 1.5 = 1.5 cereri/sec
	Allocation float64 // fractia din resursa (xi) 0.25 = 25% din CPU/RAM
}

// Sistem distribuit complet
type System struct {
	Nodes []*Node // lista de noduri
	// factor ponderare(cat de important e timpul vs costul de comunicare)
	K           float64    // K mare pune accent pe timp, K mic pe cost
	TotalLambda float64    // suma tuturor ratelor de sosire
	CostHistory []float64  // istoric costuri pentru analiza convergentei
	mut         sync.Mutex // mutex pentru acces concurent/sincronizat
}

// CreateNewSystem creeaza un sistem nou cu alocare uniforma initiala
func CreateNewSystem(lambdas []float64, mu float64, K float64) *System {
	n := len(lambdas)
	nodes := make([]*Node, n)
	totalLambda := 0.0
	for i, lambda := range lambdas {
		nodes[i] = &Node{ // initializare noduri
			ID:         i,
			Lambda:     lambda,
			Mu:         mu,
			Allocation: 1.0 / float64(n), // initializare uniforma
		}
		totalLambda += lambda
	}

	// creare si returnare sistem
	return &System{
		Nodes:       nodes,
		K:           K,
		TotalLambda: totalLambda,
		CostHistory: make([]float64, 0),
	}
}

// |=================|
// |FUNCTII DE CALCUL|
// |=================|

// ComputeCost calculeaza costul total al sistemului bazat pe costul de comunicare si timpul de raspuns
func (s *System) ComputeCost() float64 {
	totalCost := 0.0

	for _, node := range s.Nodes {
		xi := node.Allocation   // rata de resursa alocata
		lambda_i := node.Lambda // rata de sosire(trafic primit)

		// Ti = 1 / (μ - Σλ · xi)
		denominator := node.Mu - s.TotalLambda*xi // rata efectiva de sosire/numitorul
		if denominator <= 0.01 {                  // verificare stabilitate sistem
			// sistem instabil, cost infinit
			return math.Inf(1)
		}

		Ti := 1.0 / denominator // timp mediu de raspuns al nodului i

		Ci := 0.5 // cost comunicare simplificat

		// Cost = (Ci + K·Ti) · λi
		totalCost += (Ci + s.K*Ti) * lambda_i
	}
	return totalCost
}

// ComputeFirstDerivative calculeaza dU/dxi pentru nodul i
func (s *System) ComputeFirstDerivative(nodeIndex int) float64 {
	node := s.Nodes[nodeIndex]
	xi := node.Allocation // rata de resursa alocata

	denominator := node.Mu - s.TotalLambda*xi // rata efectiva de sosire/numitorul

	// dU/dxi = K · λi · Σλ / (μ - Σλ·xi)²
	derivative := s.K * node.Lambda * s.TotalLambda / (denominator * denominator)

	return derivative
}

// Compute1onSecondDerivative calculeaza ki = 1 / (d²U/dxi²) pentru nodul i
func (s *System) Compute1onSecondDerivative(nodeIndex int) float64 {
	node := s.Nodes[nodeIndex]
	xi := node.Allocation // rata de resursa alocata

	denominator := node.Mu - s.TotalLambda*xi // rata efectiva de sosire/numitorul

	// d²U/dxi² = 2·K·λi·(Σλ)² / (μ - Σλ·xi)³
	secondDerivative := 2.0 * s.K * node.Lambda *
		(s.TotalLambda * s.TotalLambda) / (denominator * denominator * denominator)

	if secondDerivative == 0 { // evitam impartirea la 0
		return 1.0
	}

	// ki = 1 / (d²U/dxi²)
	ki := 1.0 / secondDerivative
	return math.Min(ki, 5.0) // limitare pentru stabilitate
}

// Normalize asigura ca suma alocarilor este 1
func (s *System) Normalize(allocations []float64) {
	total := 0.0
	for _, allocation := range allocations {
		total += allocation
	}

	for i := range allocations {
		s.Nodes[i].Allocation = allocations[i] / total
	}
}

// alpha - rata de invatare (direct proportional cu viteza de convergenta)
// epsilon - prag de convergenta (cat de mult se apropie derivatele)

// |==========================|
// |First Derivative Algorithm|
// |==========================|
// Algoritmul incearca sa apropie alocarile incat derivatele sa fie cat mai apropiate

func FirstDerivativeAlgorithm(s *System, alpha float64, maxIter int, epsilon float64) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("First Derivative Algorithm")
	fmt.Println(strings.Repeat("=", 50))

	n := len(s.Nodes)
	for iteration := 0; iteration < maxIter; iteration++ {
		// 1. Fiecare nod calculeaza derivata in paralel
		derivatives := make([]float64, n)
		var wg sync.WaitGroup

		for i := range n {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				derivatives[index] = s.ComputeFirstDerivative(index)
			}(i)
		}
		wg.Wait()

		// 2. Calculare medie
		avgDerivative := 0.0
		for _, d := range derivatives {
			avgDerivative += d
		}
		avgDerivative /= float64(n)

		// 3. Verificare convergenta
		maxDiff := 0.0
		for _, d := range derivatives {
			diff := math.Abs(d - avgDerivative)
			if diff > maxDiff {
				maxDiff = diff
			}
		}

		if maxDiff < epsilon {
			fmt.Println("Converge la iteratia:", iteration)
			break
		}

		// 4. Actualizare alocari
		newAllocations := make([]float64, n)
		for i := range n {
			delta := -alpha * (derivatives[i] - avgDerivative)
			// delta > 0 => crestere alocare
			// delta < 0 => scadere alocare
			// delta = 0 => echilibru
			newAlloc := s.Nodes[i].Allocation + delta
			newAllocations[i] = math.Max(0.001, math.Min(0.90, newAlloc)) // evitam alocari 0
		}

		// 5. Normalizare alocari
		s.Normalize(newAllocations)

		// Aflare cost
		cost := s.ComputeCost()
		s.CostHistory = append(s.CostHistory, cost)

		if iteration%10 == 0 {
			fmt.Printf("Iter %3d: Cost = %.4f, Max diff = %.6f\n",
				iteration, cost, maxDiff)
		}
	}

	printFinalState(s)
}

// |===========================|
// |Second Derivative Algorithm|
// |===========================|
func SecondDerivativeAlgorithm(s *System, alpha float64, maxIter int, epsilon float64) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Second Derivative Algorithm")
	fmt.Println(strings.Repeat("=", 50))

	n := len(s.Nodes)

	for iteration := 0; iteration < maxIter; iteration++ {
		// 1. Calculeaza derivate si k valorile in paralel(mare=timp, mic=cost)
		derivatives := make([]float64, n)
		kValues := make([]float64, n)
		var wg sync.WaitGroup

		for i := range n {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				derivatives[index] = s.ComputeFirstDerivative(index)
				kValues[index] = s.Compute1onSecondDerivative(index) // inversul derivatei a doua = factor de scalare
			}(i)
		}
		wg.Wait()

		// 2. Calculare medie ponderata
		sumKU := 0.0
		sumK := 0.0
		for i := range n {
			sumKU += kValues[i] * derivatives[i] // d'xi/d''xi
			sumK += kValues[i]
		}
		weightedAvg := sumKU / sumK

		// 3. Verificare convergenta
		maxDiff := 0.0
		for _, d := range derivatives {
			diff := math.Abs(d - weightedAvg)
			if diff > maxDiff {
				maxDiff = diff
			}
		}

		if maxDiff < epsilon {
			fmt.Printf("Converge la iteratia: %d\n", iteration)
			break
		}

		// 4. Actualizare cu pas individual (dependent de curbura, adica k valoarea)
		newAllocations := make([]float64, n)
		for i := range n {
			delta := -alpha * kValues[i] * (derivatives[i] - weightedAvg)
			// delta > 0 => crestere alocare
			// delta < 0 => scadere alocare
			// delta = 0 => echilibru
			newAlloc := s.Nodes[i].Allocation + delta
			newAllocations[i] = math.Max(0.001, math.Min(0.90, newAlloc)) // evitam alocari 0
		}

		// Normalizare alocari
		s.Normalize(newAllocations)

		// Aflare cost
		cost := s.ComputeCost()
		s.CostHistory = append(s.CostHistory, cost)

		if iteration%5 == 0 {
			fmt.Printf("Iter %3d: Cost = %.4f, Max diff = %.6f\n",
				iteration, cost, maxDiff)
		}
	}

	printFinalState(s)
}

// Algoritm 1 vs Algoritm 2
// Pas uniform vs Pas individual
// Medie aritmetica vs Medie ponderata
// Viteza mica vs mare
// Foloseste panta vs Foloseste panta si curbura

// |==============================|
// |Pairwise Interaction Algorithm|
// |==============================|

// Reprezentare muchie intre doua noduri
type Edge struct {
	From, To int
}

func PairwiseAlgorithm(s *System, topology []Edge, alpha float64, maxIter int, epsilon float64) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Pairwise Interaction Algorithm")
	fmt.Println(strings.Repeat("=", 50))

	n := len(s.Nodes)

	for iteration := 0; iteration < maxIter; iteration++ {
		// 1. Calculeaza derivate si k valorile in paralel(mare=timp, mic=cost)
		derivatives := make([]float64, n)
		kValues := make([]float64, n)
		var wg sync.WaitGroup

		for i := range n {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				derivatives[index] = s.ComputeFirstDerivative(index)
				kValues[index] = s.Compute1onSecondDerivative(index) // inversul derivatei a doua = factor de scalare
			}(i)
		}
		wg.Wait()

		// 2. Interschimburi pe muchii
		deltas := make([]float64, n)
		for _, edge := range topology {
			i, j := edge.From, edge.To

			ki, kj := kValues[i], kValues[j]
			di, dj := derivatives[i], derivatives[j]

			// Δxi = -α · (ki·kj)/(ki+kj) · (d'i - d'j)
			exchange := -alpha * (ki * kj) / (ki + kj) * (di - dj)

			deltas[i] += exchange
			deltas[j] -= exchange
		}

		// 3. Verificare convergenta pentru vecini
		converged := true
		for _, edge := range topology {
			i, j := edge.From, edge.To
			diff := math.Abs(derivatives[i] - derivatives[j])
			if diff >= epsilon {
				converged = false
				break
			}
		}

		if converged {
			fmt.Printf("Converge la iteratia: %d\n", iteration)
			break
		}

		// 4. Actualizare alocari
		newAllocations := make([]float64, n)
		for i := range n {
			newAlloc := s.Nodes[i].Allocation + deltas[i]
			newAllocations[i] = math.Max(0.001, math.Min(0.90, newAlloc)) // evitam alocari 0
		}

		// Normalizare alocari
		s.Normalize(newAllocations)

		// Aflare cost
		cost := s.ComputeCost()
		s.CostHistory = append(s.CostHistory, cost)

		if iteration%20 == 0 {
			fmt.Printf("Iter %3d: Cost = %.4f\n",
				iteration, cost)
		}
	}

	printFinalState(s)
}

func printFinalState(s *System) {
	fmt.Println("\nAlocari finale:")
	for _, node := range s.Nodes {
		fmt.Printf("  Nod%d (λ=%.2f): x=%.3f\n",
			node.ID, node.Lambda, node.Allocation)
	}
	fmt.Printf("Cost final: %.4f", s.ComputeCost())
}

type Config struct {
	Mu      float64   `json:"mu"`
	Lambdas []float64 `json:"lambdas"`
	K       float64   `json:"K"`
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("Comparatie algoritmi de alocare a resurselor in sisteme distribuite")
	fmt.Println(strings.Repeat("=", 60))

	filename := "config.json"
	var config Config

	// configurare sistem
	configFile, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		return
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		return
	}

	mu := config.Mu
	lambdas := config.Lambdas
	K := config.K

	fmt.Printf("Noduri: %d\n", len(lambdas))
	fmt.Printf("Lambda values: %v\n", lambdas)
	fmt.Printf("μ (service rate): %.1f\n\n", mu)

	// Testare configuratii foarte precise

	// Test 1: Prima derivata
	system1 := CreateNewSystem(lambdas, mu, K)
	// simplificat pentru precizie redusa: 0.02, 200, 0.001
	FirstDerivativeAlgorithm(system1, 0.01, 1500, 0.00001)

	// Test 2: Derivata a doua
	system2 := CreateNewSystem(lambdas, mu, K)
	// simplificat pentru precizie redusa: 0.01, 100, 0.001
	SecondDerivativeAlgorithm(system2, 0.005, 1000, 0.00001)

	// Test 3: Pairwise
	system3 := CreateNewSystem(lambdas, mu, K)
	topology := []Edge{
		{0, 1}, {0, 2}, {0, 3},
		{1, 2}, {1, 3}, {2, 3},
	}
	// simplificat pentru precizie redusa: 0.05, 200, 0.001
	PairwiseAlgorithm(system3, topology, 0.02, 500, 0.00001)

	// Sumar
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Sumar comparativ al algoritmilor")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("%-20s %-12s %-12s %-8s\n",
		"Algoritm", "Iteratii", "Cost final", "Viteza")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-20s %-12d %.4f      x\n",
		"First Derivative", len(system1.CostHistory), system1.ComputeCost())
	fmt.Printf("%-20s %-12d %.4f      xxx\n",
		"Second Derivative", len(system2.CostHistory), system2.ComputeCost())
	fmt.Printf("%-20s %-12d %.4f      xx\n",
		"Pairwise", len(system3.CostHistory), system3.ComputeCost())
	fmt.Println(strings.Repeat("=", 60))

	// Generare ploturi
	systems := []*System{system1, system2, system3}
	names := []string{"First Derivative", "Second Derivative", "Pairwise"}

	// Grafic convergenta
	if err := PlotConvergence(systems, names, "plots/convergence.png"); err != nil {
		fmt.Printf("Eroare generare plot: %v\n", err)
	}

	// Grafic alocari finale
	if err := PlotAllocations(systems, names, "plots/allocations.png"); err != nil {
		fmt.Printf("Eroare generare plot: %v\n", err)
	}

	// Grafic derivate pentru verificare Nash
	if err := PlotDerivatives(systems, names, "plots/derivatives.png"); err != nil {
		fmt.Printf("Eroare generare plot: %v\n", err)
	}
}
