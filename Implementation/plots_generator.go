package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// CreateDir creeaza directorul daca nu exista
func CreateDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755) // rwx owner, rx group+others
	}
	return nil
}

// PlotConvergence genereaza grafic cu evolutia costului
func PlotConvergence(systems []*System, names []string, filename string) error {
	dir := filepath.Dir(filename)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("Eroare la creare %s: %w", dir, err)
	}

	p := plot.New()

	p.Title.Text = "Convergenta algoritmilor"
	p.X.Label.Text = "Iteratii"
	p.Y.Label.Text = "Cost total"
	p.Y.Scale = plot.LogScale{} // scala logaritmica
	p.Add(plotter.NewGrid())

	colors := []color.RGBA{
		{R: 255, A: 255},
		{G: 255, A: 255},
		{B: 255, A: 255},
	}

	for i, sys := range systems {
		pts := make(plotter.XYs, len(sys.CostHistory))
		for j, cost := range sys.CostHistory {
			pts[j].X = float64(j)
			pts[j].Y = cost
		}

		line, _ := plotter.NewLine(pts)
		line.Color = colors[i]
		line.Width = vg.Points(2)

		p.Add(line)
		p.Legend.Add(names[i], line)
	}

	p.Legend.Top = true
	p.Legend.Left = true

	return p.Save(8*vg.Inch, 6*vg.Inch, filename)
}

// PlotAllocations genereaza bar chart cu alocarile finale per nod
func PlotAllocations(systems []*System, names []string, filename string) error {
	dir := filepath.Dir(filename)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("Eroare la creare %s: %w", dir, err)
	}

	p := plot.New()

	p.Title.Text = "Alocari finale per nod"
	p.Y.Label.Text = "Alocare (xi)"
	p.X.Label.Text = "Noduri"

	w := vg.Points(20) // latime bare

	// Generare etichete pentru noduri
	nodeLabels := make([]string, len(systems[0].Nodes))
	for i := range systems[0].Nodes {
		nodeLabels[i] = fmt.Sprintf("Nod %d", i)
	}

	for i, sys := range systems {
		bars := make(plotter.Values, len(sys.Nodes))
		for j, node := range sys.Nodes {
			bars[j] = node.Allocation
		}

		bar, err := plotter.NewBarChart(bars, w)
		if err != nil {
			return err
		}
		bar.LineStyle.Width = vg.Length(0)
		bar.Color = plotutil.Color(i)
		bar.Offset = vg.Points(float64(i) * 25) // offset pentru grupare

		p.Add(bar)
		p.Legend.Add(names[i], bar)
	}

	p.Legend.Top = true
	p.NominalX(nodeLabels...)

	return p.Save(8*vg.Inch, 6*vg.Inch, filename)
}

// PlotDerivatives generează grafic cu derivatele finale (verificare Nash)
func PlotDerivatives(systems []*System, names []string, filename string) error {
	dir := filepath.Dir(filename)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("Eroare la creare %s: %w", dir, err)
	}

	p := plot.New()

	p.Title.Text = "Derivate finale (verificare Nash)"
	p.Y.Label.Text = "dU/dxi"
	p.X.Label.Text = "Noduri"

	w := vg.Points(20)

	// Generare etichete pentru noduri
	nodeLabels := make([]string, len(systems[0].Nodes))
	for i := range systems[0].Nodes {
		nodeLabels[i] = fmt.Sprintf("Nod %d", i)
	}

	for i, sys := range systems {
		derivs := make(plotter.Values, len(sys.Nodes))
		for j := range sys.Nodes {
			derivs[j] = sys.ComputeFirstDerivative(j)
		}

		bar, err := plotter.NewBarChart(derivs, w)
		if err != nil {
			return err
		}
		bar.LineStyle.Width = vg.Length(0)
		bar.Color = plotutil.Color(i)
		bar.Offset = vg.Points(float64(i) * 25)

		p.Add(bar)
		p.Legend.Add(names[i], bar)
	}

	p.Legend.Top = true
	p.NominalX(nodeLabels...)

	return p.Save(8*vg.Inch, 6*vg.Inch, filename)
}

// PlotAllocationEvolution arata cum evolueaza alocarile in timp
func PlotAllocationEvolution(sys *System, allocationHistory [][]float64, filename string) error {
	dir := filepath.Dir(filename)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("Eroare la creare %s: %w", dir, err)
	}

	p := plot.New()

	p.Title.Text = "Evolutia alocarilor in timp"
	p.X.Label.Text = "Iteratii"
	p.Y.Label.Text = "Alocare xi"
	p.Add(plotter.NewGrid())

	for nodeID := range sys.Nodes {
		pts := make(plotter.XYs, len(allocationHistory))
		for iter, allocations := range allocationHistory {
			pts[iter].X = float64(iter)
			pts[iter].Y = allocations[nodeID]
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return err
		}
		line.Color = plotutil.Color(nodeID)
		line.Width = vg.Points(1.5)
		p.Add(line)
		p.Legend.Add(fmt.Sprintf("Nod %d", nodeID), line)
	}

	p.Legend.Top = true
	p.Legend.Left = true

	return p.Save(8*vg.Inch, 6*vg.Inch, filename)
}
