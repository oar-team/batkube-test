from evalys.workload import Workload
import matplotlib.pyplot as plt
import matplotlib
import sys
import argparse

matplotlib.use('TkAgg')

parser = argparse.ArgumentParser(description="Visualisation tool for Batsim\
out_jobs.csv file")
parser.add_argument('fin', metavar='workload', type=str,
                    help='Path to the original swf file.')
parser.add_argument('fout', metavar='output', type=str,
                    help='Output destination.')
parser.add_argument('period', metavar='p', type=int,
                    help='period to extract from the workload, in hours.')
parser.add_argument('utilisation', metavar='u', type=float,
                    help='Mean utilisation whished for (between 0 and 1)')

args = parser.parse_args()

wl = Workload.from_csv(args.fin)
extracted = wl.extract_periods_with_given_utilisation(args.period, args.utilisation)

if len(extracted) == 0:
    print("could not find workload subset with given specification")
    quit()

extracted[0].to_csv(args.fout)

#extracted[0].plot()
#plt.show()
