library(ggplot2)
library(dplyr)
library(patchwork)
library(tidyr)

filename <- "200_delay170"

burst.data <- read.csv(paste("../results/max-timestep_", filename, ".csv", sep="")) %>% mutate(type="burst")

spaced.data <- read.csv(paste("../results/max-timestep_spaced_", filename, ".csv", sep="")) %>% mutate(type="spaced")

realistic.data <- read.csv("../results/max-timestep_KIT_10h_80.csv") %>% mutate(type="realistic")

csvdata <- rbind(burst.data, spaced.data, realistic.data)
colnames(csvdata)[colnames(csvdata) == "duration"] <- "simulation_time"

csvdata %>% 
	filter(type!="realistic") %>% 
	gather(-max_timestep, -type, key = "metric", value = "value") %>%
	ggplot(aes(x=max_timestep, y=value, fill=type, col=type)) +
	geom_point(alpha=0.35) +
	geom_smooth(method="loess", alpha=0.7) +
	xlab("maximum simulation timestep (ms)") +
	ylab("value (s)") +
	scale_y_continuous(trans='log10') +
	scale_x_continuous(trans='log10') +
	theme(text=element_text(size=16)) +
	facet_wrap(~metric, scales="free_y")
ggsave("../results/max-timestep_burst_sp.png", width=12, height=5)

csvdata %>% 
	filter(type=="realistic", makespan < 32560) %>% 
	gather(-max_timestep, -type, key = "metric", value = "value") %>%
	ggplot(aes(x=max_timestep, y=value, fill=type, col=type)) +
	geom_point() +
	#geom_smooth(method="loess", alpha=0.7) +
	xlab("maximum simulation timestep (ms)") +
	ylab("value (s)") +
	scale_y_continuous(trans='log10') +
	scale_x_continuous(trans='log10') +
	scale_color_viridis_d() +
	scale_fill_viridis_d() +
	theme(text=element_text(size=16)) +
	facet_wrap(~metric, scales="free_y")
ggsave("../results/max-timestep_realistic.png", width=12, height=5)

csvdata %>%
	subset(select=-max_timestep) %>%
	filter(makespan < 32560) %>% 
	gather(-simulation_time, -type, key = "metric", value = "value") %>%
	ggplot(aes(x=simulation_time, y=value, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	#geom_smooth(method="loess", alpha=0.5) +
	theme(text=element_text(size=16)) +
	scale_x_continuous(trans='log10') +
	xlab("simulation time (s)") +
	ylab("value (s)") +
	facet_wrap(~type+metric, scales="free_y")
ggsave("../results/max-timestep_sim_time_vs_metrics.png", width=12, height=8)
