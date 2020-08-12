library(ggplot2)
library(dplyr)
library(patchwork)
library(data.table)

# Script for visualizing minimal delay effect on the amount of crashes and
# simulation time

filename <- "200_delay170"

csvdata <- read.csv(paste("../results/min-delay_", filename, ".csv", sep=""))

delay <- unique(csvdata$delay)

setDT(csvdata)[, .(crash.rate = mean(exit_code),
              average.duration = mean(duration), 
	      sd.duration = sd(duration)),
          by = delay]
