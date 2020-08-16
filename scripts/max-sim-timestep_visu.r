library(ggplot2)
library(dplyr)
library(patchwork)

filename <- "200_delay170"

burst.data <- read.csv(paste("../results/max-timestep_", filename, ".csv", sep="")) %>% mutate(type="burst")

spaced.data <- read.csv(paste("../results/max-timestep_spaced_", filename, ".csv", sep="")) %>% mutate(type="spaced")

realistic.data <- read.csv("../results/max-timestep_KIT_10h_80.csv") %>% mutate(type="realistic")

csvdata <- rbind(burst.data, spaced.data, realistic.data)

#makespan <- rbind(burst.data, spaced.data, realistic.data) %>%
#	group_by(max_timestep, type) %>%
#	summarize(mean_makespan=mean(makespan),
#		  sd_makespan=sd(makespan),
#		  count=n()) %>%
#    mutate(se_makespan=mean_makespan / sqrt(count),
#           lower_ci=mean_makespan-1.96*se_makespan,
#           upper_ci=mean_makespan+1.96*se_makespan)
#
#mwt <- rbind(burst.data, spaced.data, realistic.data) %>%
#	group_by(max_timestep, type) %>%
#	summarize(mean_mwt=mean(mean_waiting_time),
#		  sd_mwt=sd(mean_waiting_time),
#		  count=n()) %>%
#    mutate(se_mwt=mean_mwt / sqrt(count),
#           lower_ci=mean_mwt-1.96*se_mwt,
#           upper_ci=mean_mwt+1.96*se_mwt)
#
#duration <- rbind(burst.data, spaced.data, realistic.data) %>%
#	group_by(max_timestep, type) %>%
#	summarize(mean_duration=mean(duration),
#		  sd_duration=sd(duration),
#		  count=n()) %>%
#    mutate(se_duration=mean_duration / sqrt(count),
#           lower_ci=mean_duration-1.96*se_duration,
#           upper_ci=mean_duration+1.96*se_duration)
#
#makespan %>% filter() %>% ggplot(aes(x=max_timestep, y=mean_makespan, fill=type, linetype=type, col=type)) +
#	#geom_point(aes(col=type)) +
#	#geom_smooth(method="loess", alpha=0.5) +
#	geom_line() +
#	geom_errorbar(aes(ymin=lower_ci, ymax=upper_ci)) +
#	ggtitle("Max timestep value effect on makespan") +
#	xlab("timeout value (ms)") +
#	ylab("mean makespan (s)") +
#	#facet_wrap(~ type) +
#	scale_x_continuous(trans='log10')
#ggsave("../results/max-timestep_makespan.png")
#
#mwt %>% filter() %>% ggplot(aes(x=max_timestep, y=mean_mwt, fill=type, linetype=type, col=type)) +
#	#geom_point(aes(col=type)) +
#	#geom_smooth(method="loess", alpha=0.5) +
#	geom_line() +
#	geom_errorbar(aes(ymax=lower_ci, ymin=upper_ci)) +
#	ggtitle("Max timestep value effect on mean waiting time") +
#	xlab("timeout value (ms)") +
#	ylab("mean waiting time (s)") +
#	#facet_wrap(~ type) +
#	scale_x_continuous(trans='log10')
#ggsave("../results/max-timestep_mwt.png")

csvdata %>% filter(type!="realistic") %>% ggplot(aes(x=max_timestep, y=makespan, fill=type, col=type)) +
	geom_point() +
	#geom_smooth(method="loess") +
	ggtitle("Maximum timestep value effect on makespan") +
	xlab("max timestep value (ms)") +
	ylab("makespan (s)") +
	scale_y_continuous(trans='log10') +
	scale_x_continuous(trans='log10')
ggsave("../results/max-timestep_makespan.png")

csvdata %>% filter() %>% ggplot(aes(x=max_timestep, y=mean_waiting_time, fill=type, col=type)) +
	geom_point() +
	#geom_smooth(method="loess") +
	ggtitle("Maximum timestep value effect on mean waiting time") +
	xlab("max timestep value (ms)") +
	ylab("mean waiting time (s)") +
	scale_x_continuous(trans='log10')
ggsave("../results/max-timestep_mwt.png")

csvdata %>% filter() %>% ggplot(aes(x=max_timestep, y=duration, fill=type, linetype=type, col=type)) +
	geom_point() +
	geom_smooth(method="loess") +
	ggtitle("Maximum timestep value effect on simulation time") +
	xlab("max timestep value value (ms)") +
	ylab("simulation time (s)") +
	scale_x_continuous(trans='log10')
ggsave("../results/max-timestep_duration.png")
