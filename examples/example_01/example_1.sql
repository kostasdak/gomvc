-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               10.4.11-MariaDB - mariadb.org binary distribution
/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- Dumping structure for table golang.products
CREATE TABLE IF NOT EXISTS `products` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `code` varchar(50) NOT NULL,
  `type` varchar(50) NOT NULL,
  `name` varchar(250) NOT NULL,
  `description` text NOT NULL,
  `price` float DEFAULT NULL,
  `images` text DEFAULT NULL,
  `status` text DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4;

-- Dumping data for table golang.products: ~3 rows (approximately)
DELETE FROM `products`;
/*!40000 ALTER TABLE `products` DISABLE KEYS */;
INSERT INTO `products` (`id`, `code`, `type`, `name`, `description`, `price`, `images`, `status`) VALUES
	(1, 'F-150', 'Truck', 'Ford F-150', 'The redesigned 2021 Ford F-150 claims one of the top spots in our full-size pickup truck rankings because of its tremendous capability and spacious, comfortable cabin', 29990, NULL, 'avail'),
	(2, 'F-250', 'Truck', 'Ford F-250', 'The all new 2022 Ford Super Duty F-250 Crew Cab comes fully ready with interior comforts, advanced technology, and exterior conveniences to make every trip you decide to go on in this powerful 4-door Pickup so enjoyable! With an available 7.3L gas V8 engine, the Super Duty F-250 Crew Cab can easily cut through any terrain and get even the toughest of jobs done! Discover how much its massive available engine can do for you. ', 35990, NULL, NULL);
/*!40000 ALTER TABLE `products` ENABLE KEYS */;


/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IFNULL(@OLD_FOREIGN_KEY_CHECKS, 1) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40111 SET SQL_NOTES=IFNULL(@OLD_SQL_NOTES, 1) */;
