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

burst.data <- read.csv(paste("../results/max-timestep_", filename, ".csv", sep=""))
burst.data$type <- "burst"
spaced.data <- read.csv(paste("../results/max-timestep_spaced_", filename, ".csv", sep=""))
spaced.data$type <- "spaced"
realistic.data <- read.csv("../results/max-timestep_KIT_10h_80.csv")
realistic.data$type <- "realistic"

csvdata <- rbind(burst.data, spaced.data, realistic.data)
#csvdata <- rbind(burst.data, spaced.data)

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
#csvdata %>% filter(type!="realistic", type!="burst") %>% ggplot(aes(x=max_timestep, y=makespan, fill=type, linetype=type, col=type)) +
#	geom_point(aes(col=type)) +
#	geom_smooth(method="loess", alpha=0.5) +
#	ggtitle("Timeout value effect on makespan") +
#	xlab("timeout value (ms)") +
#	ylab("makespan (s)") +
#	scale_x_continuous(trans='log10')
#ggsave("../results/max-timestep_makespan.png")

#mwt.sd <- sd(metrics$mwt)
#mwt.mean <- mean(metrics$mwt)
csvdata %>% filter(type!="realistic") %>% ggplot(aes(x=max_timestep, y=mean_waiting_time, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	geom_smooth(method="loess") +
	ggtitle("Maximum timestep value effect on mean waiting time") +
	xlab("max timestep value (ms)") +
	ylab("mean waiting time (s)") +
	scale_x_continuous(trans='log10')
ggsave("../results/max-timestep_mwt_without_realistic.png")

csvdata %>% filter() %>% ggplot(aes(x=max_timestep, y=duration, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	geom_smooth(method="loess") +
	ggtitle("Maximum timestep value effect on simulation time") +
	xlab("max timestep value value (ms)") +
	ylab("simulation time (s)") +
	scale_x_continuous(trans='log10')
ggsave("../results/max-timestep_duration.png")
