package frame

import "github.com/kpfaulkner/jxl-go/jxlio"

type WPParams struct {
	param1  int
	param2  int
	param3a int32
	param3b int32
	param3c int32
	param3d int32
	param3e int32
	weight  [4]int64
}

func NewWPParams(reader jxlio.BitReader) (*WPParams, error) {
	wp := WPParams{}
	var err error
	var defaultParams bool
	if defaultParams, err = reader.ReadBool(); err != nil {
		return nil, err
	}
	if defaultParams {
		wp.param1 = 16
		wp.param2 = 10
		wp.param3a = 7
		wp.param3b = 7
		wp.param3c = 7
		wp.param3d = 0
		wp.param3e = 0
		wp.weight[0] = 13
		wp.weight[1] = 12
		wp.weight[2] = 12
		wp.weight[3] = 12
	} else {
		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param1 = int(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param2 = int(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param3a = int32(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param3b = int32(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param3c = int32(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param3d = int32(param)
		}

		if param, err := reader.ReadBits(5); err != nil {
			return nil, err
		} else {
			wp.param3e = int32(param)
		}

		if data, err := reader.ReadBits(4); err != nil {
			return nil, err
		} else {
			wp.weight[0] = int64(data)
		}
		if data, err := reader.ReadBits(4); err != nil {
			return nil, err
		} else {
			wp.weight[1] = int64(data)
		}
		if data, err := reader.ReadBits(4); err != nil {
			return nil, err
		} else {
			wp.weight[2] = int64(data)
		}
		if data, err := reader.ReadBits(4); err != nil {
			return nil, err
		} else {
			wp.weight[3] = int64(data)
		}
	}

	return &wp, nil
}
