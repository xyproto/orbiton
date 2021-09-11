# Benchmarked

The quest to find a faster `bytes.Equal` function.

## Benchmark results

`equal10` does better than `bytes.Equal` for byte slices of length 0, 1, 4K, 4M and 64M.

Tested on Arch Linux, using `go version go1.17.1 linux/amd64`.

Output from `go test -bench=.`, also using benchmark functions that comes with the Go compiler source code itself:

```
goos: linux
goarch: amd64
pkg: github.com/xyproto/benchmarked
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkEqual/equal10-12         	 2065975	       585.1 ns/op
BenchmarkEqual/equal10_0-12       	646039239	         1.798 ns/op
BenchmarkEqual/equal10_1-12       	493761254	         2.335 ns/op	 428.19 MB/s
BenchmarkEqual/equal10_6-12       	257756620	         4.591 ns/op	1306.91 MB/s
BenchmarkEqual/equal10_9-12       	228489207	         5.172 ns/op	1740.02 MB/s
BenchmarkEqual/equal10_15-12      	229338196	         5.174 ns/op	2899.25 MB/s
BenchmarkEqual/equal10_16-12      	229434031	         5.171 ns/op	3094.30 MB/s
BenchmarkEqual/equal10_20-12      	208142406	         5.671 ns/op	3526.84 MB/s
BenchmarkEqual/equal10_32-12      	181841552	         6.513 ns/op	4913.11 MB/s
BenchmarkEqual/equal10_4K-12      	18256887	        65.03 ns/op	62984.27 MB/s
BenchmarkEqual/equal10_4M-12      	   14799	     86523 ns/op	48476.41 MB/s
BenchmarkEqual/equal10_64M-12     	     940	   1278144 ns/op	52504.95 MB/s
BenchmarkEqual/equal13-12         	 2220769	       527.1 ns/op
BenchmarkEqual/equal13_0-12       	577193364	         2.017 ns/op
BenchmarkEqual/equal13_1-12       	275874177	         4.332 ns/op	 230.83 MB/s
BenchmarkEqual/equal13_6-12       	274376342	         4.349 ns/op	1379.70 MB/s
BenchmarkEqual/equal13_9-12       	254787608	         4.645 ns/op	1937.70 MB/s
BenchmarkEqual/equal13_15-12      	255133258	         4.659 ns/op	3219.76 MB/s
BenchmarkEqual/equal13_16-12      	257486503	         4.620 ns/op	3463.36 MB/s
BenchmarkEqual/equal13_20-12      	223854226	         5.323 ns/op	3757.13 MB/s
BenchmarkEqual/equal13_32-12      	193529032	         6.164 ns/op	5191.49 MB/s
BenchmarkEqual/equal13_4K-12      	17835616	        66.77 ns/op	61345.96 MB/s
BenchmarkEqual/equal13_4M-12      	   16582	     72668 ns/op	57718.61 MB/s
BenchmarkEqual/equal13_64M-12     	     816	   1450565 ns/op	46263.95 MB/s
BenchmarkEqual/equal19-12         	 2160445	       543.4 ns/op
BenchmarkEqual/equal19_0-12       	573991851	         2.088 ns/op
BenchmarkEqual/equal19_1-12       	494503735	         2.469 ns/op	 404.95 MB/s
BenchmarkEqual/equal19_6-12       	259058841	         4.647 ns/op	1291.18 MB/s
BenchmarkEqual/equal19_9-12       	235608115	         5.056 ns/op	1780.16 MB/s
BenchmarkEqual/equal19_15-12      	236587689	         5.075 ns/op	2955.83 MB/s
BenchmarkEqual/equal19_16-12      	236197167	         5.021 ns/op	3186.38 MB/s
BenchmarkEqual/equal19_20-12      	204749042	         5.745 ns/op	3481.03 MB/s
BenchmarkEqual/equal19_32-12      	181522501	         6.612 ns/op	4839.40 MB/s
BenchmarkEqual/equal19_4K-12      	18206354	        66.03 ns/op	62032.45 MB/s
BenchmarkEqual/equal19_4M-12      	   15265	     79296 ns/op	52894.18 MB/s
BenchmarkEqual/equal19_64M-12     	     793	   1476543 ns/op	45449.99 MB/s
BenchmarkEqual/equal21-12         	 2014261	       593.2 ns/op
BenchmarkEqual/equal21_0-12       	573770779	         2.117 ns/op
BenchmarkEqual/equal21_1-12       	504004461	         2.469 ns/op	 404.94 MB/s
BenchmarkEqual/equal21_6-12       	237263246	         5.034 ns/op	1192.01 MB/s
BenchmarkEqual/equal21_9-12       	238259752	         5.048 ns/op	1782.87 MB/s
BenchmarkEqual/equal21_15-12      	237797186	         4.935 ns/op	3039.71 MB/s
BenchmarkEqual/equal21_16-12      	236338399	         5.109 ns/op	3131.83 MB/s
BenchmarkEqual/equal21_20-12      	211037378	         5.617 ns/op	3560.36 MB/s
BenchmarkEqual/equal21_32-12      	181614472	         6.603 ns/op	4846.61 MB/s
BenchmarkEqual/equal21_4K-12      	18281214	        70.32 ns/op	58246.41 MB/s
BenchmarkEqual/equal21_4M-12      	   13890	     85776 ns/op	48898.29 MB/s
BenchmarkEqual/equal21_64M-12     	     864	   1386072 ns/op	48416.56 MB/s
BenchmarkEqual/equal11-12         	 1904859	       647.4 ns/op
BenchmarkEqual/equal11_0-12       	637602945	         1.942 ns/op
BenchmarkEqual/equal11_1-12       	511563550	         2.372 ns/op	 421.62 MB/s
BenchmarkEqual/equal11_6-12       	233737078	         5.161 ns/op	1162.59 MB/s
BenchmarkEqual/equal11_9-12       	232003378	         5.253 ns/op	1713.24 MB/s
BenchmarkEqual/equal11_15-12      	199026771	         6.084 ns/op	2465.36 MB/s
BenchmarkEqual/equal11_16-12      	196944354	         6.126 ns/op	2611.83 MB/s
BenchmarkEqual/equal11_20-12      	180183693	         6.566 ns/op	3046.05 MB/s
BenchmarkEqual/equal11_32-12      	164495247	         7.341 ns/op	4359.37 MB/s
BenchmarkEqual/equal11_4K-12      	17147466	        74.24 ns/op	55174.97 MB/s
BenchmarkEqual/equal11_4M-12      	   10036	    119535 ns/op	35088.42 MB/s
BenchmarkEqual/equal11_64M-12     	     625	   1852017 ns/op	36235.55 MB/s
BenchmarkEqual/equal14-12         	 2092112	       584.5 ns/op
BenchmarkEqual/equal14_0-12       	553845927	         2.134 ns/op
BenchmarkEqual/equal14_1-12       	494460370	         2.416 ns/op	 413.90 MB/s
BenchmarkEqual/equal14_6-12       	243872595	         4.955 ns/op	1210.97 MB/s
BenchmarkEqual/equal14_9-12       	233283607	         5.180 ns/op	1737.46 MB/s
BenchmarkEqual/equal14_15-12      	237177555	         5.184 ns/op	2893.30 MB/s
BenchmarkEqual/equal14_16-12      	236679594	         5.158 ns/op	3102.27 MB/s
BenchmarkEqual/equal14_20-12      	213587893	         5.573 ns/op	3588.44 MB/s
BenchmarkEqual/equal14_32-12      	185139781	         6.513 ns/op	4913.14 MB/s
BenchmarkEqual/equal14_4K-12      	18088154	        71.34 ns/op	57412.14 MB/s
BenchmarkEqual/equal14_4M-12      	   10000	    102801 ns/op	40800.07 MB/s
BenchmarkEqual/equal14_64M-12     	     722	   1615388 ns/op	41543.49 MB/s
BenchmarkEqual/equal18-12         	 1924953	       620.6 ns/op
BenchmarkEqual/equal18_0-12       	566697090	         2.158 ns/op
BenchmarkEqual/equal18_1-12       	472907190	         2.497 ns/op	 400.41 MB/s
BenchmarkEqual/equal18_6-12       	222815965	         5.312 ns/op	1129.60 MB/s
BenchmarkEqual/equal18_9-12       	239056234	         5.001 ns/op	1799.72 MB/s
BenchmarkEqual/equal18_15-12      	243468238	         4.953 ns/op	3028.50 MB/s
BenchmarkEqual/equal18_16-12      	240926737	         4.962 ns/op	3224.52 MB/s
BenchmarkEqual/equal18_20-12      	207833444	         5.802 ns/op	3446.89 MB/s
BenchmarkEqual/equal18_32-12      	182235548	         6.517 ns/op	4910.27 MB/s
BenchmarkEqual/equal18_4K-12      	16972548	        69.26 ns/op	59142.68 MB/s
BenchmarkEqual/equal18_4M-12      	   10000	    103241 ns/op	40626.20 MB/s
BenchmarkEqual/equal18_64M-12     	     844	   1437415 ns/op	46687.18 MB/s
BenchmarkEqual/equal22-12         	 1898526	       630.7 ns/op
BenchmarkEqual/equal22_0-12       	567459537	         2.151 ns/op
BenchmarkEqual/equal22_1-12       	493765420	         2.471 ns/op	 404.69 MB/s
BenchmarkEqual/equal22_6-12       	194137006	         6.126 ns/op	 979.42 MB/s
BenchmarkEqual/equal22_9-12       	200525986	         6.009 ns/op	1497.70 MB/s
BenchmarkEqual/equal22_15-12      	197502253	         6.000 ns/op	2500.11 MB/s
BenchmarkEqual/equal22_16-12      	198773169	         5.962 ns/op	2683.76 MB/s
BenchmarkEqual/equal22_20-12      	179461255	         6.693 ns/op	2988.23 MB/s
BenchmarkEqual/equal22_32-12      	157300490	         7.607 ns/op	4206.79 MB/s
BenchmarkEqual/equal22_4K-12      	16677309	        72.55 ns/op	56457.79 MB/s
BenchmarkEqual/equal22_4M-12      	   13504	     88857 ns/op	47202.76 MB/s
BenchmarkEqual/equal22_64M-12     	     814	   1454392 ns/op	46142.21 MB/s
BenchmarkEqual/equal25-12         	 2134280	       574.6 ns/op
BenchmarkEqual/equal25_0-12       	551374956	         2.220 ns/op
BenchmarkEqual/equal25_1-12       	501650130	         2.463 ns/op	 406.00 MB/s
BenchmarkEqual/equal25_6-12       	332048022	         3.590 ns/op	1671.34 MB/s
BenchmarkEqual/equal25_9-12       	200107633	         6.039 ns/op	1490.23 MB/s
BenchmarkEqual/equal25_15-12      	202174316	         5.968 ns/op	2513.37 MB/s
BenchmarkEqual/equal25_16-12      	198878142	         6.045 ns/op	2646.77 MB/s
BenchmarkEqual/equal25_20-12      	180358964	         6.746 ns/op	2964.88 MB/s
BenchmarkEqual/equal25_32-12      	157352161	         7.643 ns/op	4187.03 MB/s
BenchmarkEqual/equal25_4K-12      	17715679	        73.06 ns/op	56065.31 MB/s
BenchmarkEqual/equal25_4M-12      	   10000	    102322 ns/op	40991.29 MB/s
BenchmarkEqual/equal25_64M-12     	     865	   1630404 ns/op	41160.87 MB/s
BenchmarkEqual/equal3-12          	 2051904	       586.0 ns/op
BenchmarkEqual/equal3_0-12        	549373719	         2.182 ns/op
BenchmarkEqual/equal3_1-12        	464922769	         2.671 ns/op	 374.41 MB/s
BenchmarkEqual/equal3_6-12        	337875780	         3.628 ns/op	1653.62 MB/s
BenchmarkEqual/equal3_9-12        	224038597	         5.559 ns/op	1619.12 MB/s
BenchmarkEqual/equal3_15-12       	175277424	         6.722 ns/op	2231.52 MB/s
BenchmarkEqual/equal3_16-12       	169704266	         7.077 ns/op	2260.87 MB/s
BenchmarkEqual/equal3_20-12       	146383598	         8.255 ns/op	2422.69 MB/s
BenchmarkEqual/equal3_32-12       	97067791	        12.15 ns/op	2633.98 MB/s
BenchmarkEqual/equal3_4K-12       	  950212	      1223 ns/op	3348.03 MB/s
BenchmarkEqual/equal3_4M-12       	     952	   1265423 ns/op	3314.55 MB/s
BenchmarkEqual/equal3_64M-12      	      50	  20140275 ns/op	3332.07 MB/s
BenchmarkEqual/equal29-12         	 1968086	       605.3 ns/op
BenchmarkEqual/equal29_0-12       	558403152	         2.237 ns/op
BenchmarkEqual/equal29_1-12       	475314051	         2.559 ns/op	 390.84 MB/s
BenchmarkEqual/equal29_6-12       	359692222	         3.334 ns/op	1799.51 MB/s
BenchmarkEqual/equal29_9-12       	215304590	         5.483 ns/op	1641.37 MB/s
BenchmarkEqual/equal29_15-12      	229838294	         5.296 ns/op	2832.59 MB/s
BenchmarkEqual/equal29_16-12      	206531516	         5.792 ns/op	2762.59 MB/s
BenchmarkEqual/equal29_20-12      	202702864	         5.850 ns/op	3418.83 MB/s
BenchmarkEqual/equal29_32-12      	164393308	         7.274 ns/op	4399.16 MB/s
BenchmarkEqual/equal29_4K-12      	15930814	        74.79 ns/op	54768.01 MB/s
BenchmarkEqual/equal29_4M-12      	    9844	    116397 ns/op	36034.37 MB/s
BenchmarkEqual/equal29_64M-12     	     628	   1867980 ns/op	35925.89 MB/s
BenchmarkEqual/equal30-12         	 2376816	       515.5 ns/op
BenchmarkEqual/equal30_0-12       	598743996	         1.924 ns/op
BenchmarkEqual/equal30_1-12       	473180551	         2.563 ns/op	 390.11 MB/s
BenchmarkEqual/equal30_6-12       	222978130	         5.432 ns/op	1104.61 MB/s
BenchmarkEqual/equal30_9-12       	221119120	         5.448 ns/op	1651.93 MB/s
BenchmarkEqual/equal30_15-12      	213042418	         5.764 ns/op	2602.53 MB/s
BenchmarkEqual/equal30_16-12      	211216152	         5.802 ns/op	2757.73 MB/s
BenchmarkEqual/equal30_20-12      	212192354	         5.703 ns/op	3507.19 MB/s
BenchmarkEqual/equal30_32-12      	165445788	         7.274 ns/op	4398.97 MB/s
BenchmarkEqual/equal30_4K-12      	15884214	        77.07 ns/op	53144.69 MB/s
BenchmarkEqual/equal30_4M-12      	   10224	    122706 ns/op	34181.71 MB/s
BenchmarkEqual/equal30_64M-12     	     624	   1945465 ns/op	34495.03 MB/s
BenchmarkEqual/equal1-12          	 2057217	       574.4 ns/op
BenchmarkEqual/equal1_0-12        	274793695	         4.369 ns/op
BenchmarkEqual/equal1_1-12        	234656666	         5.062 ns/op	 197.57 MB/s
BenchmarkEqual/equal1_6-12        	243128979	         5.053 ns/op	1187.32 MB/s
BenchmarkEqual/equal1_9-12        	240887617	         5.361 ns/op	1678.94 MB/s
BenchmarkEqual/equal1_15-12       	224092766	         5.382 ns/op	2787.10 MB/s
BenchmarkEqual/equal1_16-12       	227376987	         5.268 ns/op	3037.17 MB/s
BenchmarkEqual/equal1_20-12       	202382186	         5.808 ns/op	3443.38 MB/s
BenchmarkEqual/equal1_32-12       	177830709	         6.759 ns/op	4734.50 MB/s
BenchmarkEqual/equal1_4K-12       	17669605	        73.76 ns/op	55532.71 MB/s
BenchmarkEqual/equal1_4M-12       	   12747	    104840 ns/op	40006.90 MB/s
BenchmarkEqual/equal1_64M-12      	     810	   1662064 ns/op	40376.83 MB/s
BenchmarkEqual/equal2-12          	 2233320	       544.7 ns/op
BenchmarkEqual/equal2_0-12        	633440829	         1.946 ns/op
BenchmarkEqual/equal2_1-12        	490378311	         2.509 ns/op	 398.61 MB/s
BenchmarkEqual/equal2_6-12        	355749064	         3.390 ns/op	1769.80 MB/s
BenchmarkEqual/equal2_9-12        	272064630	         4.401 ns/op	2045.12 MB/s
BenchmarkEqual/equal2_15-12       	193407994	         6.216 ns/op	2413.28 MB/s
BenchmarkEqual/equal2_16-12       	180522627	         6.655 ns/op	2404.14 MB/s
BenchmarkEqual/equal2_20-12       	147143476	         8.186 ns/op	2443.20 MB/s
BenchmarkEqual/equal2_32-12       	91534165	        12.40 ns/op	2580.21 MB/s
BenchmarkEqual/equal2_4K-12       	  788373	      1470 ns/op	2786.19 MB/s
BenchmarkEqual/equal2_4M-12       	     793	   1516045 ns/op	2766.61 MB/s
BenchmarkEqual/equal2_64M-12      	      45	  24489796 ns/op	2740.28 MB/s
BenchmarkEqual/equal23-12         	 2002660	       610.7 ns/op
BenchmarkEqual/equal23_0-12       	551529040	         2.222 ns/op
BenchmarkEqual/equal23_1-12       	460201863	         2.550 ns/op	 392.21 MB/s
BenchmarkEqual/equal23_6-12       	200865837	         5.917 ns/op	1014.06 MB/s
BenchmarkEqual/equal23_9-12       	193132488	         6.317 ns/op	1424.63 MB/s
BenchmarkEqual/equal23_15-12      	196327197	         6.288 ns/op	2385.62 MB/s
BenchmarkEqual/equal23_16-12      	193361455	         6.220 ns/op	2572.52 MB/s
BenchmarkEqual/equal23_20-12      	176637307	         6.816 ns/op	2934.40 MB/s
BenchmarkEqual/equal23_32-12      	167018994	         7.379 ns/op	4336.55 MB/s
BenchmarkEqual/equal23_4K-12      	17686509	        72.21 ns/op	56719.95 MB/s
BenchmarkEqual/equal23_4M-12      	   13574	    103867 ns/op	40381.34 MB/s
BenchmarkEqual/equal23_64M-12     	     852	   1406098 ns/op	47727.02 MB/s
BenchmarkEqual/equal27-12         	 1990216	       578.2 ns/op
BenchmarkEqual/equal27_0-12       	551185255	         2.189 ns/op
BenchmarkEqual/equal27_1-12       	446652082	         2.710 ns/op	 368.97 MB/s
BenchmarkEqual/equal27_6-12       	311382925	         3.872 ns/op	1549.75 MB/s
BenchmarkEqual/equal27_9-12       	207446241	         5.663 ns/op	1589.40 MB/s
BenchmarkEqual/equal27_15-12      	140200712	         8.621 ns/op	1739.88 MB/s
BenchmarkEqual/equal27_16-12      	133544311	         9.069 ns/op	1764.18 MB/s
BenchmarkEqual/equal27_20-12      	93218812	        12.44 ns/op	1607.28 MB/s
BenchmarkEqual/equal27_32-12      	89719826	        13.19 ns/op	2425.75 MB/s
BenchmarkEqual/equal27_4K-12      	14474384	        84.80 ns/op	48304.49 MB/s
BenchmarkEqual/equal27_4M-12      	   13200	     88519 ns/op	47383.12 MB/s
BenchmarkEqual/equal27_64M-12     	     817	   1443388 ns/op	46493.98 MB/s
BenchmarkEqual/equal20-12         	 2017836	       603.8 ns/op
BenchmarkEqual/equal20_0-12       	589584826	         2.012 ns/op
BenchmarkEqual/equal20_1-12       	487533139	         2.612 ns/op	 382.79 MB/s
BenchmarkEqual/equal20_6-12       	247960797	         4.938 ns/op	1215.08 MB/s
BenchmarkEqual/equal20_9-12       	221229745	         5.363 ns/op	1678.14 MB/s
BenchmarkEqual/equal20_15-12      	222656380	         5.220 ns/op	2873.56 MB/s
BenchmarkEqual/equal20_16-12      	227234629	         5.330 ns/op	3002.14 MB/s
BenchmarkEqual/equal20_20-12      	203209803	         6.100 ns/op	3278.76 MB/s
BenchmarkEqual/equal20_32-12      	169690304	         6.891 ns/op	4643.76 MB/s
BenchmarkEqual/equal20_4K-12      	17464070	        73.19 ns/op	55963.63 MB/s
BenchmarkEqual/equal20_4M-12      	    9806	    106685 ns/op	39314.89 MB/s
BenchmarkEqual/equal20_64M-12     	     694	   1676025 ns/op	40040.49 MB/s
BenchmarkEqual/equal26-12         	 1919725	       634.1 ns/op
BenchmarkEqual/equal26_0-12       	536709402	         2.299 ns/op
BenchmarkEqual/equal26_1-12       	453884952	         2.566 ns/op	 389.69 MB/s
BenchmarkEqual/equal26_6-12       	353573353	         3.412 ns/op	1758.26 MB/s
BenchmarkEqual/equal26_9-12       	187069982	         6.241 ns/op	1441.99 MB/s
BenchmarkEqual/equal26_15-12      	189691318	         6.283 ns/op	2387.55 MB/s
BenchmarkEqual/equal26_16-12      	188946277	         6.275 ns/op	2549.73 MB/s
BenchmarkEqual/equal26_20-12      	174762952	         7.053 ns/op	2835.63 MB/s
BenchmarkEqual/equal26_32-12      	154852246	         7.762 ns/op	4122.68 MB/s
BenchmarkEqual/equal26_4K-12      	17132721	        74.24 ns/op	55169.39 MB/s
BenchmarkEqual/equal26_4M-12      	   13024	     91877 ns/op	45651.16 MB/s
BenchmarkEqual/equal26_64M-12     	     728	   1463016 ns/op	45870.21 MB/s
BenchmarkEqual/equal32-12         	 2275591	       513.0 ns/op
BenchmarkEqual/equal32_0-12       	554559524	         2.250 ns/op
BenchmarkEqual/equal32_1-12       	495515328	         2.510 ns/op	 398.37 MB/s
BenchmarkEqual/equal32_6-12       	359572626	         3.404 ns/op	1762.57 MB/s
BenchmarkEqual/equal32_9-12       	196536649	         6.240 ns/op	1442.42 MB/s
BenchmarkEqual/equal32_15-12      	207833025	         5.867 ns/op	2556.56 MB/s
BenchmarkEqual/equal32_16-12      	178983472	         6.703 ns/op	2387.12 MB/s
BenchmarkEqual/equal32_20-12      	181960868	         6.633 ns/op	3015.28 MB/s
BenchmarkEqual/equal32_32-12      	151683027	         7.888 ns/op	4056.81 MB/s
BenchmarkEqual/equal32_4K-12      	15889996	        78.24 ns/op	52353.18 MB/s
BenchmarkEqual/equal32_4M-12      	    8884	    118775 ns/op	35313.03 MB/s
BenchmarkEqual/equal32_64M-12     	     627	   1947938 ns/op	34451.24 MB/s
BenchmarkEqual/equal8-12          	 1824740	       659.4 ns/op
BenchmarkEqual/equal8_0-12        	552632442	         2.221 ns/op
BenchmarkEqual/equal8_1-12        	479686189	         2.536 ns/op	 394.37 MB/s
BenchmarkEqual/equal8_6-12        	201091162	         6.040 ns/op	 993.43 MB/s
BenchmarkEqual/equal8_9-12        	202152032	         6.036 ns/op	1491.07 MB/s
BenchmarkEqual/equal8_15-12       	202062627	         5.804 ns/op	2584.26 MB/s
BenchmarkEqual/equal8_16-12       	204755218	         5.923 ns/op	2701.32 MB/s
BenchmarkEqual/equal8_20-12       	181489239	         6.643 ns/op	3010.80 MB/s
BenchmarkEqual/equal8_32-12       	159619744	         7.491 ns/op	4271.73 MB/s
BenchmarkEqual/equal8_4K-12       	17823728	        71.36 ns/op	57402.72 MB/s
BenchmarkEqual/equal8_4M-12       	   13144	    105555 ns/op	39735.58 MB/s
BenchmarkEqual/equal8_64M-12      	     801	   1492590 ns/op	44961.35 MB/s
BenchmarkEqual/equal9-12          	 1935068	       628.6 ns/op
BenchmarkEqual/equal9_0-12        	550921977	         2.234 ns/op
BenchmarkEqual/equal9_1-12        	475943481	         2.588 ns/op	 386.44 MB/s
BenchmarkEqual/equal9_6-12        	209831638	         5.690 ns/op	1054.43 MB/s
BenchmarkEqual/equal9_9-12        	213169746	         5.787 ns/op	1555.20 MB/s
BenchmarkEqual/equal9_15-12       	206530141	         5.614 ns/op	2672.10 MB/s
BenchmarkEqual/equal9_16-12       	211459522	         5.739 ns/op	2788.07 MB/s
BenchmarkEqual/equal9_20-12       	183999735	         6.286 ns/op	3181.72 MB/s
BenchmarkEqual/equal9_32-12       	161338754	         7.262 ns/op	4406.28 MB/s
BenchmarkEqual/equal9_4K-12       	14932940	        80.17 ns/op	51091.66 MB/s
BenchmarkEqual/equal9_4M-12       	   10000	    104150 ns/op	40271.76 MB/s
BenchmarkEqual/equal9_64M-12      	     807	   1661499 ns/op	40390.55 MB/s
BenchmarkEqual/equal12-12         	 2090760	       591.3 ns/op
BenchmarkEqual/equal12_0-12       	565653589	         2.204 ns/op
BenchmarkEqual/equal12_1-12       	459807460	         2.606 ns/op	 383.78 MB/s
BenchmarkEqual/equal12_6-12       	225118838	         5.201 ns/op	1153.52 MB/s
BenchmarkEqual/equal12_9-12       	227518706	         5.203 ns/op	1729.67 MB/s
BenchmarkEqual/equal12_15-12      	225478806	         5.377 ns/op	2789.56 MB/s
BenchmarkEqual/equal12_16-12      	227784796	         5.185 ns/op	3085.54 MB/s
BenchmarkEqual/equal12_20-12      	200941242	         6.007 ns/op	3329.22 MB/s
BenchmarkEqual/equal12_32-12      	173048186	         6.876 ns/op	4654.16 MB/s
BenchmarkEqual/equal12_4K-12      	16775028	        73.28 ns/op	55894.38 MB/s
BenchmarkEqual/equal12_4M-12      	   10000	    103823 ns/op	40398.46 MB/s
BenchmarkEqual/equal12_64M-12     	     811	   1432532 ns/op	46846.32 MB/s
BenchmarkEqual/equal15-12         	 1962676	       616.8 ns/op
BenchmarkEqual/equal15_0-12       	546088586	         2.186 ns/op
BenchmarkEqual/equal15_1-12       	461351565	         2.575 ns/op	 388.35 MB/s
BenchmarkEqual/equal15_6-12       	212932554	         5.651 ns/op	1061.68 MB/s
BenchmarkEqual/equal15_9-12       	211951327	         5.729 ns/op	1571.01 MB/s
BenchmarkEqual/equal15_15-12      	206942436	         5.633 ns/op	2662.78 MB/s
BenchmarkEqual/equal15_16-12      	212279516	         5.647 ns/op	2833.12 MB/s
BenchmarkEqual/equal15_20-12      	188833741	         6.452 ns/op	3099.65 MB/s
BenchmarkEqual/equal15_32-12      	164767618	         7.468 ns/op	4284.89 MB/s
BenchmarkEqual/equal15_4K-12      	17267283	        74.99 ns/op	54623.69 MB/s
BenchmarkEqual/equal15_4M-12      	   12980	     90369 ns/op	46413.00 MB/s
BenchmarkEqual/equal15_64M-12     	     828	   1456535 ns/op	46074.32 MB/s
BenchmarkEqual/equal6-12          	 1785670	       647.7 ns/op
BenchmarkEqual/equal6_0-12        	547757719	         2.238 ns/op
BenchmarkEqual/equal6_1-12        	475142102	         2.599 ns/op	 384.76 MB/s
BenchmarkEqual/equal6_6-12        	203082556	         5.892 ns/op	1018.40 MB/s
BenchmarkEqual/equal6_9-12        	203587292	         5.845 ns/op	1539.69 MB/s
BenchmarkEqual/equal6_15-12       	202268768	         6.025 ns/op	2489.44 MB/s
BenchmarkEqual/equal6_16-12       	198577626	         5.821 ns/op	2748.53 MB/s
BenchmarkEqual/equal6_20-12       	177985912	         6.673 ns/op	2997.07 MB/s
BenchmarkEqual/equal6_32-12       	162141235	         7.558 ns/op	4233.67 MB/s
BenchmarkEqual/equal6_4K-12       	17053588	        72.20 ns/op	56735.14 MB/s
BenchmarkEqual/equal6_4M-12       	   12699	     94495 ns/op	44386.29 MB/s
BenchmarkEqual/equal6_64M-12      	     847	   1672120 ns/op	40134.01 MB/s
BenchmarkEqual/equal16-12         	 2001782	       594.3 ns/op
BenchmarkEqual/equal16_0-12       	547169810	         2.311 ns/op
BenchmarkEqual/equal16_1-12       	461083999	         2.644 ns/op	 378.23 MB/s
BenchmarkEqual/equal16_6-12       	244786725	         4.953 ns/op	1211.31 MB/s
BenchmarkEqual/equal16_9-12       	225462790	         5.304 ns/op	1696.99 MB/s
BenchmarkEqual/equal16_15-12      	223193098	         5.397 ns/op	2779.15 MB/s
BenchmarkEqual/equal16_16-12      	227713274	         5.364 ns/op	2982.72 MB/s
BenchmarkEqual/equal16_20-12      	197663150	         6.084 ns/op	3287.22 MB/s
BenchmarkEqual/equal16_32-12      	169859264	         7.084 ns/op	4517.22 MB/s
BenchmarkEqual/equal16_4K-12      	17209856	        71.99 ns/op	56895.33 MB/s
BenchmarkEqual/equal16_4M-12      	   13398	    106031 ns/op	39557.37 MB/s
BenchmarkEqual/equal16_64M-12     	     836	   1449509 ns/op	46297.67 MB/s
BenchmarkEqual/equal17-12         	 1898793	       632.8 ns/op
BenchmarkEqual/equal17_0-12       	530376230	         2.260 ns/op
BenchmarkEqual/equal17_1-12       	458112337	         2.548 ns/op	 392.50 MB/s
BenchmarkEqual/equal17_6-12       	222666117	         5.426 ns/op	1105.76 MB/s
BenchmarkEqual/equal17_9-12       	233696464	         5.109 ns/op	1761.49 MB/s
BenchmarkEqual/equal17_15-12      	238958382	         5.038 ns/op	2977.25 MB/s
BenchmarkEqual/equal17_16-12      	231243774	         5.049 ns/op	3169.14 MB/s
BenchmarkEqual/equal17_20-12      	207571822	         5.775 ns/op	3463.15 MB/s
BenchmarkEqual/equal17_32-12      	179976402	         6.720 ns/op	4762.09 MB/s
BenchmarkEqual/equal17_4K-12      	16628649	        73.04 ns/op	56078.19 MB/s
BenchmarkEqual/equal17_4M-12      	   13059	    105777 ns/op	39652.15 MB/s
BenchmarkEqual/equal17_64M-12     	     831	   1460989 ns/op	45933.86 MB/s
BenchmarkEqual/equal24-12         	 2333318	       525.8 ns/op
BenchmarkEqual/equal24_0-12       	557877853	         2.238 ns/op
BenchmarkEqual/equal24_1-12       	478615189	         2.406 ns/op	 415.63 MB/s
BenchmarkEqual/equal24_6-12       	192294768	         6.255 ns/op	 959.25 MB/s
BenchmarkEqual/equal24_9-12       	184022047	         6.613 ns/op	1360.91 MB/s
BenchmarkEqual/equal24_15-12      	181190649	         6.537 ns/op	2294.66 MB/s
BenchmarkEqual/equal24_16-12      	180481687	         6.657 ns/op	2403.48 MB/s
BenchmarkEqual/equal24_20-12      	163736740	         7.352 ns/op	2720.20 MB/s
BenchmarkEqual/equal24_32-12      	145829623	         8.112 ns/op	3944.60 MB/s
BenchmarkEqual/equal24_4K-12      	16276156	        75.65 ns/op	54146.86 MB/s
BenchmarkEqual/equal24_4M-12      	   12650	     92103 ns/op	45539.26 MB/s
BenchmarkEqual/equal24_64M-12     	     828	   1447662 ns/op	46356.72 MB/s
BenchmarkEqual/equal31-12         	 2293537	       527.9 ns/op
BenchmarkEqual/equal31_0-12       	532020832	         2.195 ns/op
BenchmarkEqual/equal31_1-12       	500196176	         2.456 ns/op	 407.11 MB/s
BenchmarkEqual/equal31_6-12       	329668524	         3.672 ns/op	1634.06 MB/s
BenchmarkEqual/equal31_9-12       	198342458	         6.139 ns/op	1465.95 MB/s
BenchmarkEqual/equal31_15-12      	176712624	         6.766 ns/op	2216.82 MB/s
BenchmarkEqual/equal31_16-12      	180881635	         6.638 ns/op	2410.35 MB/s
BenchmarkEqual/equal31_20-12      	178604072	         6.643 ns/op	3010.48 MB/s
BenchmarkEqual/equal31_32-12      	157151064	         7.663 ns/op	4175.93 MB/s
BenchmarkEqual/equal31_4K-12      	15580176	        79.39 ns/op	51591.45 MB/s
BenchmarkEqual/equal31_4M-12      	    9834	    123010 ns/op	34097.22 MB/s
BenchmarkEqual/equal31_64M-12     	     607	   1974978 ns/op	33979.55 MB/s
BenchmarkEqual/bytes.Equal-12     	 2070726	       604.9 ns/op
BenchmarkEqual/bytes.Equal_0-12   	245286045	         4.911 ns/op
BenchmarkEqual/bytes.Equal_1-12   	258280090	         4.722 ns/op	 211.77 MB/s
BenchmarkEqual/bytes.Equal_6-12   	257432746	         4.639 ns/op	1293.43 MB/s
BenchmarkEqual/bytes.Equal_9-12   	242185982	         4.950 ns/op	1818.05 MB/s
BenchmarkEqual/bytes.Equal_15-12  	245456658	         5.008 ns/op	2995.38 MB/s
BenchmarkEqual/bytes.Equal_16-12  	245295454	         4.845 ns/op	3302.61 MB/s
BenchmarkEqual/bytes.Equal_20-12  	211978808	         5.707 ns/op	3504.24 MB/s
BenchmarkEqual/bytes.Equal_32-12  	185558857	         6.586 ns/op	4858.89 MB/s
BenchmarkEqual/bytes.Equal_4K-12  	17545779	        73.34 ns/op	55847.14 MB/s
BenchmarkEqual/bytes.Equal_4M-12  	   12546	     91582 ns/op	45798.24 MB/s
BenchmarkEqual/bytes.Equal_64M-12 	     784	   1655015 ns/op	40548.79 MB/s
BenchmarkEqual/equal4-12          	 1886408	       632.7 ns/op
BenchmarkEqual/equal4_0-12        	531992954	         2.236 ns/op
BenchmarkEqual/equal4_1-12        	457104560	         2.566 ns/op	 389.76 MB/s
BenchmarkEqual/equal4_6-12        	206207140	         5.766 ns/op	1040.63 MB/s
BenchmarkEqual/equal4_9-12        	206623597	         5.827 ns/op	1544.51 MB/s
BenchmarkEqual/equal4_15-12       	211698676	         5.747 ns/op	2609.84 MB/s
BenchmarkEqual/equal4_16-12       	207511932	         5.747 ns/op	2783.92 MB/s
BenchmarkEqual/equal4_20-12       	185520783	         6.292 ns/op	3178.71 MB/s
BenchmarkEqual/equal4_32-12       	165416799	         7.312 ns/op	4376.43 MB/s
BenchmarkEqual/equal4_4K-12       	17639560	        75.56 ns/op	54211.95 MB/s
BenchmarkEqual/equal4_4M-12       	   13080	    104524 ns/op	40127.57 MB/s
BenchmarkEqual/equal4_64M-12      	     716	   1679129 ns/op	39966.48 MB/s
BenchmarkEqual/equal7-12          	 1893801	       626.3 ns/op
BenchmarkEqual/equal7_0-12        	543417753	         2.247 ns/op
BenchmarkEqual/equal7_1-12        	470599206	         2.631 ns/op	 380.15 MB/s
BenchmarkEqual/equal7_6-12        	234525284	         5.031 ns/op	1192.60 MB/s
BenchmarkEqual/equal7_9-12        	211740754	         5.685 ns/op	1583.04 MB/s
BenchmarkEqual/equal7_15-12       	209109124	         5.771 ns/op	2599.03 MB/s
BenchmarkEqual/equal7_16-12       	211092248	         5.757 ns/op	2779.30 MB/s
BenchmarkEqual/equal7_20-12       	194715068	         6.350 ns/op	3149.78 MB/s
BenchmarkEqual/equal7_32-12       	166014380	         7.241 ns/op	4419.07 MB/s
BenchmarkEqual/equal7_4K-12       	16104822	        71.49 ns/op	57291.84 MB/s
BenchmarkEqual/equal7_4M-12       	   10000	    105234 ns/op	39856.76 MB/s
BenchmarkEqual/equal7_64M-12      	     715	   1448766 ns/op	46321.40 MB/s
BenchmarkEqual/equal28-12         	 1784074	       659.3 ns/op
BenchmarkEqual/equal28_0-12       	544866654	         2.202 ns/op
BenchmarkEqual/equal28_1-12       	469725586	         2.581 ns/op	 387.52 MB/s
BenchmarkEqual/equal28_6-12       	173001136	         6.824 ns/op	 879.20 MB/s
BenchmarkEqual/equal28_9-12       	174287700	         6.921 ns/op	1300.37 MB/s
BenchmarkEqual/equal28_15-12      	174433114	         6.848 ns/op	2190.37 MB/s
BenchmarkEqual/equal28_16-12      	174987211	         6.928 ns/op	2309.39 MB/s
BenchmarkEqual/equal28_20-12      	177164593	         6.799 ns/op	2941.77 MB/s
BenchmarkEqual/equal28_32-12      	142274325	         8.324 ns/op	3844.39 MB/s
BenchmarkEqual/equal28_4K-12      	15412558	        77.46 ns/op	52881.26 MB/s
BenchmarkEqual/equal28_4M-12      	    9817	    124402 ns/op	33715.64 MB/s
BenchmarkEqual/equal28_64M-12     	     625	   1950624 ns/op	34403.80 MB/s
PASS
ok  	github.com/xyproto/benchmarked	640.899s
```

## Equal function

Here's the "equal10" function:

```go
func equal10(a, b []byte) bool {
    la := len(a)
    lb := len(b)
    switch la {
    case 2:
        return lb == 2 && a[0] == b[0] && a[1] == b[1]
    case 1:
        return lb == 1 && a[0] == b[0]
    case 0:
        return lb == 0
    case lb:
        return !(string(a) != string(b))
    default:
        return false
    }
}
```

## bytes.Equal

For comparison, `bytes.Equal` looks like [this](https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/bytes/bytes.go;l=18):

```go
func Equal(a, b []byte) bool {
    return string(a) == string(b)
}
```

## Accuracy

I am aware that perfect benchmarking is a tricky.

Please let me know if you have improvements to how the functions are benchmarked, or how the benchmarks are interpreted!


## General info

* Version: 0.2.0
* License: BSD
