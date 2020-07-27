from evalys.jobset import JobSet
import matplotlib.pyplot as plt
import argparse

parser = argparse.ArgumentParser(description="Visualisation tool for Batsim\
out_jobs.csv file")
parser.add_argument('file', metavar='file', type=str,
                    help='path to the csv file.')
args = parser.parse_args()

js = JobSet.from_csv(args.file)
js.plot(with_details=True)
plt.show()
