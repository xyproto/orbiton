package colour

import "math"

type LinearTransferFunction struct {
}

func (tf LinearTransferFunction) ToLinear(input float64) float64 {
	return input
}

func (tf LinearTransferFunction) FromLinear(input float64) float64 {
	return input
}

type SRGBTransferFunction struct {
}

func (tf SRGBTransferFunction) ToLinear(input float64) float64 {
	if input < 0.0404482362771082 {
		return input * 0.07739938080495357
	}
	return math.Pow((input+0.055)*0.9478672985781991, 2.4)
}

func (tf SRGBTransferFunction) FromLinear(input float64) float64 {
	if input < 0.00313066844250063 {
		return input * 12.92
	}
	return 1.055*math.Pow(input, 0.4166666666666667) - 0.055
}

type BT709TransferFunction struct {
}

func (tf BT709TransferFunction) ToLinear(input float64) float64 {
	if input < 0.081242858298635133011 {
		return input * 0.22222222222222222222
	}

	return math.Pow((input+0.0992968268094429403)*0.90967241568627260377, 2.2222222222222222222)
}

func (tf BT709TransferFunction) FromLinear(input float64) float64 {
	if input < 0.018053968510807807336 {
		return 4.5 * input
	}
	return 1.0992968268094429403*math.Pow(input, 0.45) - 0.0992968268094429403
}

type PQTransferFunction struct {
}

func (tf PQTransferFunction) ToLinear(input float64) float64 {
	d := math.Pow(input, 0.012683313515655965121)
	return math.Pow((d-0.8359375)/(18.8515625+18.6875*d), 6.2725880551301684533)
}

func (tf PQTransferFunction) FromLinear(input float64) float64 {
	d := math.Pow(input, 0.159423828125)
	return math.Pow((0.8359375+18.8515625*d)/(1.0+18.6875*d), 78.84375)
}

type GammaTransferFunction struct {
	gamma        float64
	inverseGamma float64
}

func NewGammaTransferFunction(transfer int32) GammaTransferFunction {
	gtf := GammaTransferFunction{}
	gtf.gamma = 1e-7 * float64(transfer)
	gtf.inverseGamma = 1e7 / float64(transfer)
	return gtf
}

func (tf GammaTransferFunction) ToLinear(input float64) float64 {
	return math.Pow(input, tf.inverseGamma)
}

func (tf GammaTransferFunction) FromLinear(input float64) float64 {
	return math.Pow(input, tf.gamma)
}
