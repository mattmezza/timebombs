package sample

/*
TIMEBOMB(2025-11-15): Rip out the feature flag scaffolding.

	We shipped the experiment, the flag is always-on now, but there's
	still branching logic everywhere in the checkout flow.
*/
func Checkout() {}
