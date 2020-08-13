library(ggplot2)
library(dplyr)
library(patchwork)
library(data.table)

# Script for visualizing minimal delay effect on the amount of crashes and
# simulation time

filename <- "spaced_200_delay170"

csvdata <- read.csv(paste("../results/min-delay_", filename, ".csv", sep=""))
old.csvdata <- read.csv(paste("../results/min-delay_", filename, "-aggregated.csv", sep=""))
old.csvdata$crash.rate <- sapply(old.csvdata$success_rate, function(x){return(1-x)})

delay <- unique(csvdata$delay)

aggregated <- setDT(csvdata)[, .(crash.rate = mean(exit_code),
				 average.duration = mean(duration), 
				 med.duration = median(duration),
				 sd.duration = sd(duration),
				 max.duration = max(duration),
				 min.duration = min(duration)),
by = delay]

old.csvdata %>% ggplot(aes(x=delay, y=crash.rate)) +
	geom_point() +
	geom_line() +
	ylim(0, 1) +
	ggtitle("Minimum delay value effect on crash rate (spaced workload)") +
	xlab("delay value (ms)") +
	ylab("crash rate")
ggsave(paste("../results/", "min-delay_", filename, "_crash_old", ".png", sep=""))

aggregated %>% ggplot(aes(x=delay, y=crash.rate)) +
	geom_point() +
	geom_line() +
	ylim(0, 1) +
	ggtitle("Minimum delay value effect on crash rate (spaced workload)") +
	xlab("delay value (ms)") +
	ylab("crash rate")
ggsave(paste("../results/", "min-delay_", filename, "_crash", ".png", sep=""))

csvdata %>% ggplot(aes(x=as.factor(delay), y=duration)) + 
    geom_boxplot(fill="slateblue", alpha=0.2) + 
	xlab("delay value (ms)") +
	ylab("duration")
ggsave(paste("../results/", "min-delay_", filename, "_duration", ".png", sep=""))

old.csvdata %>% filter(mean_success_sim_time>0) %>%
	ggplot(aes(x=delay, y=mean_success_sim_time)) + 
	geom_line() +
	geom_point() +
	xlab("delay value (ms)") +
	ggtitle("Minimum delay value effect on duration (spaced workload)") +
	ylab("mean duration")
ggsave(paste("../results/", "min-delay_", filename, "_duration_old", ".png", sep=""))
