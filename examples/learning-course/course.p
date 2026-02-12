
wanna-learn:
	- [ ]  Learn probability and stats and betting maths
	- [ ]  what is a distribution? what is a random variable?
		- [ ]  To reason statistically, you need a stable distribution. Black Swans live in **unknown or shifting distributions**. You don’t know:
			
			– the tail thickness,
			
			– the true variance,
			
			– whether the distribution tomorrow matches today.
			
		
		So statements like “this happens once every 100 years” are category errors. You are inferring frequency from a sample that *cannot contain the event you’re worried about*.
		
	- [ ]  what is a joint distribution? what does that mean?
	- [ ]  what is a hilbert space?
	- [ ]  what are the different camps of probability - induction vs. bayesian - can you explain/teach them?
	- [ ]  how does a p-test work? why?
	- [ ]  how does measuring more times increase accuracy?
	- [ ]  what is a fat tailed distribution? like what’s the equation
	- [ ]  what’s **sampling, what’s bayesian statistics, what’s a monte carlo method?**
		- [ ]  https://en.wikipedia.org/wiki/Monte_Carlo_method
	- [ ]  what’s a fourier transform?



; @thought-shapes.p
; map(all-shapes, wanna-learn)

; analyse these topics and sketch a minimal course outline that logically proceeds through teaching to completion all of these ideas.
; the course outline is a simple skeleton index with a few sub items per item.
; @wanna-learn


course-outline:
	## 1. Foundations: Random Variables & Distributions
	- What is a random variable? (discrete vs continuous)
	- What is a probability distribution? (PMF, PDF, CDF)
	- Key examples: uniform, normal, binomial, Poisson
	- Parameters: mean, variance, moments

	## 2. Joint Distributions & Dependencies
	- Joint, marginal, and conditional distributions
	- Independence vs correlation vs causation
	- Covariance and correlation coefficients
	- Bayes' theorem as a bridge (first exposure)

	## 3. Schools of Probability
	- Frequentist (classical/inductive) interpretation
	- Bayesian interpretation: priors, likelihoods, posteriors
	- Where they agree, where they diverge, and why it matters

	; ## 4. Sampling & the Law of Large Numbers
	; - What is sampling? (populations vs samples)
	; - Why measuring more times increases accuracy (LLN, CLT)
	; - Standard error and convergence rates
	; - Bias, variance, and the tradeoff

	; ## 5. Frequentist Inference: Hypothesis Testing
	; - What is a p-test (p-value)? How and why it works
	; - Null vs alternative hypotheses
	; - Type I / Type II errors, significance levels
	; - Common misinterpretations

	; ## 6. Bayesian Statistics & Computational Methods
	; - Bayesian updating in practice
	; - Prior selection and sensitivity
	; - Monte Carlo methods: using randomness to estimate deterministic quantities
	; - Markov Chain Monte Carlo (MCMC) basics

	; ## 7. Fat Tails, Black Swans & the Limits of Inference
	; - Thin-tailed vs fat-tailed distributions (Gaussian vs Pareto/power-law)
	; - The maths: power-law exponent α, when variance is infinite
	; - Why stable distributions are required for standard stats to work
	; - Black Swans: unknown/shifting distributions, non-ergodicity
	; - The category error: inferring rare-event frequency from insufficient samples

	; ## 8. Betting Maths & Decision Under Uncertainty
	; - Expected value, edge, and odds
	; - Kelly criterion for optimal bet sizing
	; - Ruin theory and why fat tails break naïve EV calculations
	; - Practical takeaway: when to trust your model and when not to

	; ## 9. Inner Product Spaces & the Fourier Transform
	; - What is a Hilbert space? (inner products, orthogonality, completeness)
	; - Why it matters: random variables *live* in a Hilbert space (L²)
	; - What is a Fourier transform? Decomposing signals into frequencies
	; - Connection: characteristic functions — the Fourier transform of a probability distribution


context-problem:
	For this topic area, explain the original context of the problem it solved. 
	The thing people often leave out in explaining/teaching is the context.
	They describe simply what we have now without the struggle of invention to find it.
	When that is often the most interesting story, and the rationale which connects it to reality.

book(topic):
	topic -> lessons (map(lessons, context-problem))

map(course-outline, context-problem)