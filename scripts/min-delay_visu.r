library(ggplot2)
library(dplyr)
library(patchwork)
library(data.table)

# Script for visualizing minimal delay effect on the amount of crashes and
# simulation time

filename <- "200_delay170"

spaced.data <- read.csv(paste("../results/min-delay_spaced_", filename, ".csv", sep="")) %>% mutate(type="spaced")
burst.data <- read.csv(paste("../results/min-delay_", filename, ".csv", sep="")) %>% mutate(type="burst")
realistic.data <- read.csv("../results/min-delay_KIT_10h_80.csv") %>% mutate(type="realistic")

csvdata <- rbind(spaced.data, burst.data, realistic.data) %>%
	filter(exit_code==0) %>%
	group_by(delay, type) %>%
	summarize(mean_duration=mean(duration),
		  sd_duration=sd(duration),
		  count=n()) %>%
    mutate(se_duration=mean_duration / sqrt(count),
           lower_ci=mean_duration-1.96*se_duration,
           upper_ci=mean_duration+1.96*se_duration)

old.data <- read.csv(paste("../results/min-delay_spaced_", filename, "-aggregated.csv", sep=""))
old.data$crash.rate <- sapply(old.data$success_rate, function(x){return(1-x)})

csvdata %>% ggplot(aes(x=delay, y=mean_duration, fill=type, col=type)) + 
	geom_line() +
	#geom_ribbon(alpha=0.5) +
	geom_errorbar(aes(ymax=lower_ci, ymin=upper_ci)) +
	geom_vline(aes(xintercept=20, col="timeout"), linetype="dashed") +
	xlab("delay value (ms)") +
	ylab("mean simulation time (s)") +
	theme_bw(base_size=18) +
	facet_wrap(~ type)
ggsave("../results/min-delay_duration.png", width=12, height=5)

old.data %>% ggplot(aes(x=delay, y=crash.rate)) +
	geom_point() +
	geom_line() +
	ylim(0, 1) +
	theme_bw(base_size=18) +
	xlab("delay value (ms)") +
	ylab("crash rate")
ggsave("../results/min-delay_crash_old.png")

#old.data %>% filter(mean_success_sim_time>0) %>%
#	ggplot(aes(x=delay, y=mean_success_sim_time)) + 
#	geom_line() +
#	geom_point() +
#	theme(base_size=16) +
#	xlab("delay value (ms)") +
#	ylab("mean duration")
#ggsave(paste("../results/", "min-delay_", filename, "_duration_old", ".png", sep=""))
