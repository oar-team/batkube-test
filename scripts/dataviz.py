from evalys.jobset import JobSet
import matplotlib.pyplot as plt

js = JobSet.from_csv("../../batkube/expe-out/out_jobs.csv")
js.plot(with_details=True)
plt.show()