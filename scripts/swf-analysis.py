from evalys.workload import Workload
import matplotlib.pyplot as plt
import matplotlib
import sys

matplotlib.use('TkAgg')

wl = Workload.from_csv("swf/NASA-iPSC-1993-3.1-cln.swf")
extracted = wl.extract_periods_with_given_utilisation(20, 0.7)

extracted[0].to_csv("workload.swf")



