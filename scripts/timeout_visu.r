library(ggplot2)
library(dplyr)
library(patchwork)
library(tidyr)

#compute.metrics <- function(filename) {
#	data <- read.csv(filename)
#	makespan <- max(data$finish_time)
#	mean_waiting_time <- mean(data$waiting_time)
#	return(list("makespan" = makespan, "mean_waiting_time" = mean_waiting_time))
#}

burst.data <- read.csv("../results/timeout_200_delay170.csv")
burst.data$type <- "burst"
spaced.data <- read.csv("../results/timeout_spaced_200_delay170.csv")
spaced.data$type <- "spaced"
realistic.data <- read.csv("../results/timeout_KIT_10h_80.csv")
realistic.data$type <- "realistic"

csvdata <- rbind(burst.data, spaced.data, realistic.data)
colnames(csvdata)[colnames(csvdata) == "duration"] <- "simulation_time"


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
csvdata %>%
	filter(type != "realistic", makespan < 5000) %>% 
	#pivot_longer(-timeout:-type, names_to = "metric", values_to = "value")
	gather(-timeout, -type, key = "metric", value = "value") %>%
	#pivot_longer(timeout)
	ggplot(aes(x=timeout, y=value, col=type, fill=type)) +
	geom_point(aes(col=type, fill=type)) +
	geom_smooth(method="loess", alpha=0.5) +
	theme(text=element_text(size=16)) +
	xlab("timeout (ms)") +
	ylab("value (s)") +
	#scale_color_viridis_d() +
	#scale_fill_viridis_d() +
	facet_wrap(~metric, scales="free_y")
ggsave("../results/timeout_burst_spaced.png", width=12, height=5)

csvdata %>%
	filter(type == "realistic", makespan < 50000) %>% 
	gather(-timeout, -type, key = "metric", value = "value") %>%
	ggplot(aes(x=timeout, y=value, fill=type, linetype=type, col=type)) +
	geom_point(aes(col=type)) +
	#geom_smooth(method="loess", alpha=0.5) +
	theme(text=element_text(size=16)) +
	scale_color_viridis_d() +
	scale_fill_viridis_d() +
	xlab("timeout (ms)") +
	ylab("value (s)") +
	facet_wrap(~metric, scales="free_y")
ggsave("../results/timeout_realistic.png", width=12, height=5)
