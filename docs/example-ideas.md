Here are quasi-practical “this feels like a thing you’d actually use” examples that are small enough to run on a simulator, but real enough to force the right capabilities. I’m assuming a **gate-model** simulator (circuits + measurements). I’ll call out when something needs density matrices / noise / mid-circuit measurement, etc.

## 1) Bell-pair “network handshake” + entanglement verification

**What you do:** Create a Bell pair, measure in different bases, estimate correlations, compute CHSH value.
**Why it’s quasi-practical:** This is basically the unit test for “can my simulator model entanglement and basis changes,” plus it mirrors how you’d validate an entanglement link in a network.

**Circuit sketch:**

* Prepare |00⟩
* H on q0, CNOT(q0→q1)
* Randomly choose measurement bases (Z or X) for each qubit: apply H before measuring if X
* Accumulate counts; estimate correlation E(a,b); compute CHSH S

**Capabilities required**

* Statevector simulation (2 qubits)
* Single- and two-qubit gates: H, CNOT
* Measurement in computational basis
* Ability to do **basis change** (X-basis measurement via H + Z-measure)
* Sampling many shots; basic post-processing of outcomes

---

## 2) Quantum teleportation with feed-forward (mid-circuit measurement)

**What you do:** Teleport an arbitrary single-qubit state |ψ⟩ from Alice to Bob using an EPR pair, measuring Alice’s qubits and classically controlling corrections on Bob.

**Why quasi-practical:** Teleportation is the archetype for “quantum + classical control flow,” which shows up everywhere (error correction, repeaters, adaptive algorithms).

**Circuit sketch:**

* Prepare |ψ⟩ on q0 (use random Bloch angles)
* Prepare Bell pair between q1 and q2
* CNOT(q0→q1), H(q0)
* Measure q0, q1 → classical bits m0, m1
* If m1==1 apply X to q2; if m0==1 apply Z to q2
* Verify q2 ≈ |ψ⟩ (state fidelity or tomography)

**Capabilities required**

* **Mid-circuit measurement**
* **Classical register** + conditional gates (feed-forward)
* Single-qubit arbitrary rotations (or at least RX/RY/RZ) to create |ψ⟩
* Either: access to final statevector to compute fidelity, **or** tomography support (repeated basis measurements)

---

## 3) Superdense coding (2 classical bits via 1 qubit)

**What you do:** Alice encodes 2 classical bits into one qubit using an entangled pair, sends it to Bob, Bob decodes with CNOT+H and measures.

**Why quasi-practical:** It’s basically “entanglement-assisted communication” in toy form, and it’s a clean correctness check for entanglement + local Paulis.

**Circuit sketch:**

* Share Bell pair (q0 with Alice, q1 with Bob)
* Alice encodes bits (a,b) with Z^a X^b on q0
* Bob decodes: CNOT(q0→q1), H(q0)
* Measure both; should recover (a,b)

**Capabilities required**

* H, CNOT, X, Z
* Measurement + shot statistics
* Ability to model “sending a qubit” can just be relabeling ownership—no special sim feature needed

---

## 4) Phase kickback as a “quantum sensor” toy

**What you do:** Use a control qubit to pick up phase from a unitary on a target, then read it out via interference. (This is the core phenomenon behind phase estimation.)

**Why quasi-practical:** Phase kickback is what makes quantum metrology and many algorithms tick; it’s a great way to verify your simulator handles global vs relative phase correctly.

**Circuit sketch:**

* Put control in |+⟩
* Apply controlled-U on target prepared in an eigenstate of U (e.g., U=RZ(θ), eigenstate |0⟩)
* Interfere with H on control, measure; probability depends on θ

**Capabilities required**

* Controlled single-qubit unitaries (controlled-RZ or controlled-U)
* Correct complex amplitudes / relative phase handling
* Measurement & sampling

---

## 5) Quantum Phase Estimation (QPE) on a tiny unitary

**What you do:** Estimate the eigenphase of a known unitary (like RZ(θ)) using a few ancillas.

**Why quasi-practical:** This is a “real algorithm,” and it forces you to implement controlled powers of U and the inverse QFT.

**Circuit sketch:**

* t ancillas in |+⟩, one target in eigenstate
* Apply controlled-U^{2^k} from ancilla k
* Inverse QFT on ancillas
* Measure ancillas → phase bits

**Capabilities required**

* Multi-qubit circuits (say 4–8 qubits)
* Controlled-U^{2^k} (either built-in “power” or repeated application)
* QFT / inverse QFT decomposition into H + controlled phase rotations
* Good numerical stability (phases are unforgiving)

---

## 6) Grover search for a “practical-ish” constraint satisfaction toy

**What you do:** Solve a tiny SAT-like constraint: find a bitstring that satisfies a predicate (e.g., “x has exactly two 1s” or “(x0 XOR x1) AND x2 is true”).

**Why quasi-practical:** Grover is the canonical “search/optimization primitive.” Even toy instances force you to support an oracle and diffusion.

**Circuit sketch:**

* n qubits uniform superposition
* Oracle: flip phase of “good” states (often via multi-controlled Z with ancilla)
* Diffusion operator
* Repeat ~⌊π/4 √(N/M)⌋ times
* Measure

**Capabilities required**

* Multi-controlled gates (Toffoli / MCX / MCZ), or ability to decompose them
* Ancilla qubits and uncomputation
* Reasonable sampling and probability inspection
* Optional: ability to inspect amplitudes to verify amplification

---

## 7) Variational Quantum Eigensolver (VQE) on H₂ (2–4 qubits)

**What you do:** Use a parameterized ansatz, measure expectation values of a small Hamiltonian, and run a classical optimizer to minimize energy.

**Why it’s quasi-practical:** This is the most “near-term real” workload historically studied: hybrid quantum/classical loops, lots of measurements, expectation estimation.

**What you can do without chemistry plumbing:** Use a **toy Hamiltonian** (sum of Pauli strings) first, then swap in published H₂ coefficients later.

**Capabilities required**

* Parameterized circuits (RX/RY/RZ with symbolic parameters)
* Ability to run the same circuit many times with different parameters
* Expectation estimation of Pauli strings via basis changes (measure X/Y/Z)
* Classical outer loop integration (you can keep this outside the simulator API, but you need fast repeated runs)
* Optional but very helpful: **shot noise** simulation and/or exact expectation from statevector

---

## 8) Quantum Approximate Optimization Algorithm (QAOA) for MaxCut on a small graph

**What you do:** Encode a graph (say 3–6 nodes), apply p layers of QAOA (cost Hamiltonian + mixer), sample bitstrings, evaluate cut values.

**Why quasi-practical:** This is the poster child for “combinatorial optimization with NISQ-ish circuits.”

**Capabilities required**

* Two-qubit ZZ interactions (or CNOT + RZ + CNOT decomposition)
* Parameterized angles (γ, β)
* Sampling + classical scoring
* Optional: support for transpilation to a native gate set

---

## 9) Noise-aware demo: repetition code or simple bit-flip channel

**What you do:** Show that encoding a logical bit into 3 qubits and majority-voting reduces error under a bit-flip noise model.

**Why quasi-practical:** If you ever want to compare to hardware behavior, you need noise. Also, it tests density matrices / Kraus operators.

**Capabilities required (choose one approach)**

* **Density-matrix simulation** *or* stochastic “quantum trajectories”
* Noise channels: X with probability p (bit-flip), optionally depolarizing
* Measurement and classical post-processing (majority vote)
* Optional: mid-circuit syndrome measurement if you go beyond the simplest version

---

## 10) Randomized benchmarking-lite (gate quality sanity check)

**What you do:** Generate random Clifford sequences, append the inverse, measure survival probability. With noise, you get an exponential decay.

**Why quasi-practical:** This is how people characterize devices. In sim-land, it becomes a strong test of your gate composition correctness (and noise model if present).

**Capabilities required**

* Ability to sample random circuits
* Clifford gate set (H, S, CNOT is enough with compilation)
* Inversion/composition correctness
* Optional: noise model + fitting support

---

# A minimal capability ladder (so you can stage development)

**Stage A: “Born rule works”**

* Statevector, gates {X,Z,H,CNOT}, end-of-circuit measurement, shots

**Stage B: “Basis + phase aren’t lying”**

* RZ (or generic single-qubit rotations), controlled-phase, basis-change measurement

**Stage C: “Adaptive circuits”**

* Mid-circuit measurement + classical control

**Stage D: “Algorithms and workflows”**

* Parameter binding + fast reruns, expectation values of Pauli strings, simple transpilation

**Stage E: “Hardware realism”**

* Noise (Kraus / density matrix or trajectories), optionally reset, leakage (later)

---

## Recommendation: build toward this trio first

1. **Bell/CHSH** (entanglement + basis correctness)
2. **Teleportation** (mid-circuit measurement + classical control)
3. **VQE toy Hamiltonian** (parameter binding + expectation estimation + performance under many runs)

That set forces the exact features most simulators trip over: phases, measurement semantics, and repeated execution speed.

If you share your intended API shape (Python? Rust? Go? “circuit as AST” vs “imperative builder”), I can map each example into a concrete test harness style (golden tests + property tests) without turning it into a science fair project.
