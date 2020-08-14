library(ggplot2)
library(dplyr)
library(patchwork)
library(data.table)

# Script for visualizing minimal delay effect on the amount of crashes and
# simulation time

filename <- "200_delay170"

spaced.data <- read.csv(paste("../results/min-delay_spaced_", filename, ".csv", sep="")) %>% mutate(type="spaced")
#spaced.data %>% filter(spaced.data, exit_code==0)
#spaced.aggregated <- setDT(spaced.data)[, .(avg.duration = mean(duration),
#					    med.duration = median(duration),
#					    max.duration = max(duration),
#					    min.duration = min(duration)),
#by = delay]
#spaced.aggregated$type <- "spaced"
#
burst.data <- read.csv(paste("../results/min-delay_", filename, ".csv", sep="")) %>% mutate(type="burst")
#burst.data <- filter(burst.data, exit_code==0)
#burst.aggregated <- setDT(burst.data)[, .(avg.duration = mean(duration),
#					  med.duration = median(duration),
#					  max.duration = max(duration),
#					  min.duration = min(duration)),
#by = delay]
#burst.aggregated$type <- "burst"
#
realistic.data <- read.csv("../results/min-delay_KIT_10h_80.csv") %>% mutate(type="realistic")
#realistic.data <- filter(realistic.data, exit_code==0)
#realistic.aggregated <- setDT(realistic.data)[, .(avg.duration = mean(duration),
#					  med.duration = median(duration),
#					  max.duration = max(duration),
#					  min.duration = min(duration)),
#by = delay]
#realistic.aggregated$type <- "realistic"

# csvdata <- rbind(spaced.aggregated, burst.aggregated, realistic.aggregated)
csvdata <- rbind(spaced.data, burst.data, realistic.data) %>%
	filter(exit_code==0) %>%
	group_by(delay, type) %>%
	summarize(mean_duration=mean(duration),
		  sd_duration=sd(duration),
		  count=n()) %>%
    mutate(se_duration=mean_duration / sqrt(count),
           lower_ci=mean_duration-1.96*se_duration,
           upper_ci=mean_duration+1.96*se_duration)

old.data <- read.csv(paste("../results/min-delay_", filename, "-aggregated.csv", sep=""))
old.data$crash.rate <- sapply(old.data$success_rate, function(x){return(1-x)})

#delay <- unique(csvdata$delay)

csvdata %>% ggplot(aes(x=delay, y=mean_duration, fill=type, col=type)) + 
	geom_line() +
	#geom_ribbon(alpha=0.5) +
	geom_errorbar(aes(ymax=lower_ci, ymin=upper_ci)) +
	geom_vline(aes(xintercept=20, col="timeout"), linetype="dashed") +
	xlab("delay value (ms)") +
	ggtitle("Minimum delay value effect on duration") +
	ylab("mean duration") +
	theme(aspect.ratio=1) +
	facet_wrap(~ type)
ggsave("../results/min-delay_duration.png")

old.data %>% ggplot(aes(x=delay, y=crash.rate)) +
	geom_point() +
	geom_line() +
	ylim(0, 1) +
	ggtitle("Minimum delay value effect on crash rate (spaced workload)") +
	xlab("delay value (ms)") +
	ylab("crash rate")
ggsave(paste("../results/", "min-delay_", filename, "_crash_old", ".png", sep=""))

# aggregated %>% ggplot(aes(x=delay, y=crash.rate)) +
# 	geom_point() +
# 	geom_line() +
# 	ylim(0, 1) +
# 	ggtitle("Minimum delay value effect on crash rate (spaced workload)") +
# 	xlab("delay value (ms)") +
# 	ylab("crash rate")
# ggsave(paste("../results/", "min-delay_", filename, "_crash", ".png", sep=""))

# csvdata %>% ggplot(aes(x=as.factor(delay), y=duration)) + 
#     geom_boxplot(fill="slateblue", alpha=0.2) + 
# 	xlab("delay value (ms)") +
# 	ylab("duration")
# ggsave(paste("../results/", "min-delay_", filename, "_duration", ".png", sep=""))


old.data %>% filter(mean_success_sim_time>0) %>%
	ggplot(aes(x=delay, y=mean_success_sim_time)) + 
	geom_line() +
	geom_point() +
	xlab("delay value (ms)") +
	ggtitle("Minimum delay value effect on duration (spaced workload)") +
	ylab("mean duration")
ggsave(paste("../results/", "min-delay_", filename, "_duration_old", ".png", sep=""))
