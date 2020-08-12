library(ggplot2)
library(dplyr)
library(patchwork)

compute.metrics <- function(filename) {
	data <- read.csv(filename)
	makespan <- max(data$finish_time)
	mean_waiting_time <- mean(data$waiting_time)
	return(list("makespan" = makespan, "mean_waiting_time" = mean_waiting_time))
}

filename <- "spaced_200_delay170"
type <- "timeout"

csvdata <- read.csv(paste("../results/timeout_", filename, ".csv", sep=""))

makespan <- c()
mwt <- c()

for (i in 0:9) {
	met = compute.metrics(paste("../results/", filename, "_", i, "_jobs.csv", sep=""))
	makespan <- append(makespan, met$makespan)
	mwt <- append(mwt, met$mean_waiting_time)
}

metrics <- data.frame(makespan, mwt)

makespan.sd <- sd(metrics$makespan)
makespan.mean <- mean(metrics$makespan)
csvdata[-1:-3,] %>% ggplot(aes(x=timeout, y=makespan)) +
	geom_point() +
	geom_hline(yintercept=makespan.mean+makespan.sd, linetype="dashed") +
	geom_hline(yintercept=makespan.mean) +
	geom_hline(yintercept=makespan.mean-makespan.sd, linetype="dashed") +
	ggtitle("Timeout value effect on makespan (spaced workload)") +
	xlab("timeout value (ms)") +
	ylab("makespan (s)")
ggsave(paste("../results/", type, "_", filename, "_makespan", ".png", sep=""))

mwt.sd <- sd(metrics$mwt)
mwt.mean <- mean(metrics$mwt)
csvdata[-1:-3,] %>% ggplot(aes(x=timeout, y=mean_waiting_time)) +
	geom_point() +
	geom_hline(yintercept=mwt.mean+mwt.sd, linetype="dashed") +
	geom_hline(yintercept=mwt.mean) +
	geom_hline(yintercept=mwt.mean-mwt.sd, linetype="dashed") +
	ggtitle("Timeout value effect on mean waiting time (spaced workload)") +
	xlab("timeout value (ms)") +
	ylab("mean waiting time (s)")
ggsave(paste("../results/", type, "_", filename, "_mwt", ".png", sep=""))

csvdata[-1:-3,] %>% ggplot(aes(x=timeout, y=duration)) +
	geom_point() +
	geom_smooth(method="loess") +
	ggtitle("Timeout value effect on simulation time (spaced workload)") +
	xlab("timeout value (ms)") +
	ylab("simulation time (s)")
ggsave(paste("../results/", type, "_", filename, "_duration", ".png", sep=""))
