package gfx

func ExampleCmplxSin() {
	Log("%f %f %f",
		CmplxSin(complex(1, 2)),
		CmplxSin(complex(2, 3)),
		CmplxSin(complex(4, 5)),
	)

	// Output:
	// (3.165779+1.959601i) (9.154499-4.168907i) (-56.162274-48.502455i)
}

func ExampleCmplxSinh() {
	Log("%f %f %f",
		CmplxSinh(complex(1, 2)),
		CmplxSinh(complex(2, 3)),
		CmplxSinh(complex(4, 5)),
	)

	// Output:
	// (-0.489056+1.403119i) (-3.590565+0.530921i) (7.741118-26.186527i)
}

func ExampleCmplxCos() {
	Log("%f %f %f",
		CmplxCos(complex(1, 2)),
		CmplxCos(complex(2, 3)),
		CmplxCos(complex(4, 5)),
	)

	// Output:
	// (2.032723-3.051898i) (-4.189626-9.109228i) (-48.506859+56.157175i)
}

func ExampleCmplxCosh() {
	Log("%f %f %f",
		CmplxCosh(complex(1, 2)),
		CmplxCosh(complex(2, 3)),
		CmplxCosh(complex(4, 5)),
	)

	// Output:
	// (-0.642148+1.068607i) (-3.724546+0.511823i) (7.746313-26.168964i)
}

func ExampleCmplxTan() {
	Log("%f %f %f",
		CmplxTan(complex(1, 2)),
		CmplxTan(complex(2, 3)),
		CmplxTan(complex(4, 5)),
	)

	// Output:
	// (0.033813+1.014794i) (-0.003764+1.003239i) (0.000090+1.000013i)
}

func ExampleCmplxTanh() {
	Log("%f %f %f",
		CmplxTanh(complex(1, 2)),
		CmplxTanh(complex(2, 3)),
		CmplxTanh(complex(4, 5)),
	)

	// Output:
	// (1.166736-0.243458i) (0.965386-0.009884i) (1.000563-0.000365i)
}

func ExampleCmplxPow() {
	Log("%f %f",
		CmplxPow(complex(1, 2), complex(2, 3)),
		CmplxPow(complex(4, 5), complex(5, 6)),
	)

	// Output:
	// (-0.015133-0.179867i) (-49.591090+4.323851i)
}

func ExampleCmplxSqrt() {
	Log("%f %f %f",
		CmplxSqrt(complex(1, 2)),
		CmplxSqrt(complex(2, 3)),
		CmplxSqrt(complex(4, 5)),
	)

	// Output:
	// (1.272020+0.786151i) (1.674149+0.895977i) (2.280693+1.096158i)
}

func ExampleCmplxPhase() {
	Log("%f %f %f",
		CmplxPhase(complex(1, 2)),
		CmplxPhase(complex(2, 3)),
		CmplxPhase(complex(4, 5)),
	)

	// Output:
	// 1.107149 0.982794 0.896055
}
