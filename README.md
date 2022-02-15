# GCL Playground
A development environment for constructing programs calculationally using Dijkstra's techniques.

### Architecture

GCL Playground is composed of four primary entities which communicate via messages, illustrated below.

```mermaid
sequenceDiagram
actor Programmer
participant SpecDisplay
participant Refiner
participant LemmaDisplay
participant LemmaEditor

activate Programmer

Programmer ->> Refiner: Specify (precondition, postcondition)
activate Refiner
Refiner ->> SpecDisplay: append (spec)
deactivate Refiner

loop until fully refined

alt refine

Programmer ->> SpecDisplay: Select (subprogram)
activate SpecDisplay
SpecDisplay ->> Refiner: refine (subprogram)
activate Refiner
Refiner ->> LemmaEditor: getLemma (obligation)
activate LemmaEditor
Programmer ->> LemmaEditor: Write (lemma)
LemmaEditor ->> LemmaDisplay: append (lemma)
LemmaEditor ->> Refiner: Done
deactivate LemmaEditor
Refiner ->> SpecDisplay: append (spec)
deactivate Refiner
deactivate SpecDisplay

else undo

Programmer ->> SpecDisplay: Undo
SpecDisplay ->> LemmaDisplay: undo

else edit lemma

Programmer ->> LemmaDisplay: Edit (id)
activate LemmaDisplay
LemmaDisplay ->> LemmaEditor: edit (id, lemma)
activate LemmaEditor
Programmer ->> LemmaEditor: Edit (lemma)
LemmaEditor ->> LemmaDisplay: update (id, lemma)
deactivate LemmaEditor
deactivate LemmaDisplay

end

end

deactivate Programmer
```
