from evalys.workload import Workload
import matplotlib.pyplot as plt
import matplotlib
import sys

matplotlib.use('TkAgg')

wl = Workload.from_csv("swf/NASA-iPSC-1993-3.1-cln.swf")
extracted = wl.extract_periods_with_given_utilisation(10, 0.7)

wl.to_csv("workload.swf")



