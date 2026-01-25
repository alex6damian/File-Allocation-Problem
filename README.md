# File Allocation Problem - Abordare Microeconomică

Proiect de cercetare pentru cursul de Sisteme Distribuite care abordează problema alocării optime a resurselor într-un sistem distribuit folosind teoria microeconomică.

## Descriere

Am implementat și comparat 3 algoritmi descentralizați pentru alocarea resurselor în sisteme distribuite:
- **First Derivative Algorithm** - gradient descent clasic
- **Second Derivative Algorithm** - metoda Newton cu convergență accelerată
- **Pairwise Interaction Algorithm** - complet descentralizat, comunicare doar între vecini

Fiecare nod din sistem se comportă ca un agent economic care negociază cu celelalte noduri pentru a ajunge la un echilibru Nash, minimizând costul total (întârziere + comunicare).

## Structură Repository

- `documentatie.tex` - documentația completă a proiectului (teorie + analiză)
- `Implementation/` - codul sursă în Go
- `prezentare.tex` - prezentare Beamer pentru susținerea proiectului
