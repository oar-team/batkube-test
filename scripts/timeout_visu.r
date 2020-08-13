library(ggplot2)
library(dplyr)
library(patchwork)

compute.metrics <- function(filename) {
	data <- read.csv(filename)
	makespan <- max(data$finish_time)
	mean_waiting_time <- mean(data$waiting_time)
	return(list("makespan" = makespan, "mean_waiting_time" = mean_waiting_time))
}

filename <- "200_delay170"
type <- "timeout"

burst.data <- read.csv(paste("../results/timeout_", filename, ".csv", sep=""))
burst.data$type <- "burst"
spaced.data <- read.csv(paste("../results/timeout_spaced_", filename, ".csv", sep=""))
spaced.data$type <- "spaced"
#realistic.data <- read.csv("../results/timeout_KIT_10h_80.csv")
#realistic.data$type <- "realistic"

#csvdata <- rbind(burst.data, spaced.data, realistic.data)
csvdata <- rbind(burst.data, spaced.data)

#makespan <- c()
#mwt <- c()
#for (i in 0:9) {
#	met = compute.metrics(paste("../results/", filename, "_", i, "_jobs.csv", sep=""))
#	makespan <- append(makespan, met$makespan)
#	mwt <- append(mwt, met$mean_waiting_time)
#}

#metrics <- data.frame(makespan, mwt)
#
#makespan.sd <- sd(metrics$makespan)
#makespan.mean <- mean(metrics$makespan)
csvdata %>% filter(timeout<=100, makespan <5000) %>% ggplot(aes(x=timeout, y=makespan, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	geom_smooth(method="loess", alpha=0.5) +
	#geom_hline(yintercept=makespan.mean+makespan.sd, linetype="dashed") +
	#geom_hline(yintercept=makespan.mean) +
	#geom_hline(yintercept=makespan.mean-makespan.sd, linetype="dashed") +
	ggtitle("Timeout value effect on makespan") +
	xlab("timeout value (ms)") +
	ylab("makespan (s)")
ggsave("../results/timeout_makespan.png")

#mwt.sd <- sd(metrics$mwt)
#mwt.mean <- mean(metrics$mwt)
csvdata %>% filter(timeout<=100, mean_waiting_time<3000) %>% ggplot(aes(x=timeout, y=mean_waiting_time, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	geom_smooth(method="loess") +
	#geom_hline(yintercept=mwt.mean+mwt.sd, linetype="dashed") +
	#geom_hline(yintercept=mwt.mean) +
	#geom_hline(yintercept=mwt.mean-mwt.sd, linetype="dashed") +
	ggtitle("Timeout value effect on mean waiting time") +
	xlab("timeout value (ms)") +
	ylab("mean waiting time (s)")
ggsave("../results/timeout_mwt.png")

csvdata[-1:-3,] %>% filter(timeout<=100) %>% ggplot(aes(x=timeout, y=duration, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	geom_smooth(method="loess") +
	ggtitle("Timeout value effect on simulation time") +
	xlab("timeout value (ms)") +
	ylab("simulation time (s)")
ggsave("../results/timeout_duration.png")
